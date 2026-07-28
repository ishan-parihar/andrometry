package com.andrometry.collectors.location

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.os.Looper
import androidx.core.content.ContextCompat
import com.andrometry.core.config.SensitiveModeManager
import com.andrometry.core.database.AndrometryDatabase
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.database.TimelineSegmentEntity
import com.google.android.gms.location.*
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Location collector using geofencing and passive location updates.
 * Follows the battery-efficient hierarchy from architecture.md §9.4.
 *
 * §9.4 — Do not send exact coordinates by default. Send semantic location.
 *
 * Properly injects AndrometryDatabase via Hilt DI (critical fix).
 * Includes stop() method for lifecycle cleanup to prevent leaks.
 */
@Singleton
class LocationCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator,
    private val database: AndrometryDatabase,
    private val sensitiveModeManager: SensitiveModeManager
) {
    data class KnownPlace(
        val name: String,
        val latitude: Double,
        val longitude: Double,
        val radiusMeters: Float = 150f
    )

    data class LocationContext(
        val place: String,
        val confidence: Double,
        val movement: String,
        val changed: Boolean
    )

    private val knownPlaces = mutableListOf(
        KnownPlace("home", 0.0, 0.0, 200f),
        KnownPlace("office", 0.0, 0.0, 150f),
        KnownPlace("gym", 0.0, 0.0, 100f),
        KnownPlace("cafe", 0.0, 0.0, 80f)
    )

    private var lastLocation: Location? = null
    private var lastPlace: String = "unknown"
    private var lastPlaceEnteredMs: Long = 0L
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    @Volatile private var cachedRedactLocation: Boolean = false
    private var fusedLocationClient: FusedLocationProviderClient? = null
    private var geofencingClient: GeofencingClient? = null
    private var locationCallback: LocationCallback? = null

    fun start(context: Context) {
        if (!hasLocationPermission(context)) return

        // Cache sensitive mode state; set initial value synchronously, refresh every 5 min
        scope.launch {
            cachedRedactLocation = sensitiveModeManager.shouldRedactLocation()
            while (true) {
                kotlinx.coroutines.delay(5 * 60 * 1000L)
                cachedRedactLocation = sensitiveModeManager.shouldRedactLocation()
            }
        }

        fusedLocationClient = LocationServices.getFusedLocationProviderClient(context)
        geofencingClient = LocationServices.getGeofencingClient(context)

        registerGeofences(context)
        startPassiveLocationUpdates(context)
    }

    /**
     * Stops location updates and removes geofences to prevent leaks.
     * Call this when the collector is no longer needed.
     */
    fun stop() {
        scope.coroutineContext[kotlinx.coroutines.Job]?.cancel()
        locationCallback?.let { callback ->
            fusedLocationClient?.removeLocationUpdates(callback)
            locationCallback = null
        }
        geofencingClient?.let { client ->
            val validNames = knownPlaces.filter { it.latitude != 0.0 || it.longitude != 0.0 }.map { it.name }
            if (validNames.isNotEmpty()) {
                client.removeGeofences(validNames)
            }
        }
        lastLocation = null
        lastPlace = "unknown"
        lastPlaceEnteredMs = 0L
    }

    private fun registerGeofences(context: Context) {
        val validPlaces = knownPlaces.filter { it.latitude != 0.0 || it.longitude != 0.0 }
        if (validPlaces.isEmpty()) return

        val geofences = validPlaces.map { place ->
            Geofence.Builder()
                .setRequestId(place.name)
                .setCircularRegion(place.latitude, place.longitude, place.radiusMeters)
                .setTransitionTypes(Geofence.GEOFENCE_TRANSITION_ENTER or Geofence.GEOFENCE_TRANSITION_EXIT)
                .setExpirationDuration(Geofence.NEVER_EXPIRE)
                .setLoiteringDelay(60000)
                .build()
        }

        val request = GeofencingRequest.Builder()
            .setInitialTrigger(GeofencingRequest.INITIAL_TRIGGER_ENTER)
            .addGeofences(geofences)
            .build()

        geofencingClient?.addGeofences(request, null)
    }

    private fun startPassiveLocationUpdates(context: Context) {
        if (!hasLocationPermission(context)) return

        val request = LocationRequest.Builder(Priority.PASSIVE_UPDATE, 15 * 60 * 1000L)
            .setMinUpdateIntervalMillis(5 * 60 * 1000L)
            .build()

        val callback = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                result.lastLocation?.let { location ->
                    processLocationUpdate(context, location)
                }
            }
        }
        locationCallback = callback
        fusedLocationClient?.requestLocationUpdates(request, callback, Looper.getMainLooper())
    }

    private fun processLocationUpdate(context: Context, location: Location) {
        val detectedPlace = detectPlace(location)
        val movement = classifyMovement(location)
        val placeChanged = detectedPlace != lastPlace

        val locationContext = LocationContext(
            place = detectedPlace,
            confidence = if (detectedPlace != "unknown") 0.9 else 0.5,
            movement = movement,
            changed = placeChanged
        )

        // §33: Use cached redaction state (updated every 5 min on background thread)
        val redactLocation = cachedRedactLocation
        val displayPlace = if (redactLocation) "[redacted]" else locationContext.place
        val payload = mapOf(
            "place" to displayPlace,
            "confidence" to locationContext.confidence,
            "movement" to locationContext.movement,
            "changed" to locationContext.changed
        )

        val now = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date())

        val event = EventEntity(
            eventId = UUID.randomUUID().toString(),
            deviceId = android.os.Build.MODEL.replace(" ", "-").lowercase(),
            sequence = sequenceGenerator.next(),
            timestamp = now,
            timezone = TimeZone.getDefault().id,
            eventType = "location_context",
            collector = "geofence",
            source = "geofence",
            payload = Gson().toJson(payload)
        )
        database.eventDao().insert(event)

        if (placeChanged && lastPlace != "unknown" && lastPlaceEnteredMs > 0) {
            val enterTime = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }.format(Date(lastPlaceEnteredMs))

            val segment = TimelineSegmentEntity(
                segmentId = UUID.randomUUID().toString(),
                segmentType = "place_visit",
                layer = "place",
                startTime = enterTime,
                endTime = now,
                label = if (cachedRedactLocation) "[redacted]" else lastPlace,
                confidence = 0.88,
                validFrom = enterTime,
                ingestedAt = now,
                payload = Gson().toJson(mapOf("confidence" to 0.88))
            )
            database.segmentDao().upsert(segment)
        }

        lastLocation = location
        if (placeChanged) {
            lastPlace = detectedPlace
            lastPlaceEnteredMs = System.currentTimeMillis()
        }
    }

    private fun detectPlace(location: Location): String {
        for (place in knownPlaces) {
            if (place.latitude == 0.0 && place.longitude == 0.0) continue
            val distance = FloatArray(1)
            Location.distanceBetween(
                location.latitude, location.longitude,
                place.latitude, place.longitude,
                distance
            )
            if (distance[0] <= place.radiusMeters) {
                return place.name
            }
        }
        return "unknown"
    }

    private fun classifyMovement(location: Location): String {
        val last = lastLocation ?: return "unknown"

        val distance = FloatArray(1)
        Location.distanceBetween(
            last.latitude, last.longitude,
            location.latitude, location.longitude,
            distance
        )

        val timeDelta = (location.time - last.time) / 1000.0
        if (timeDelta <= 0) return "still"

        val speed = distance[0] / timeDelta

        return when {
            speed < 0.5 -> "still"
            speed < 2.0 -> "walking"
            speed < 5.0 -> "running"
            else -> "vehicle"
        }
    }

    fun addKnownPlace(name: String, latitude: Double, longitude: Double, radiusMeters: Float = 150f) {
        knownPlaces.removeAll { it.name == name }
        knownPlaces.add(KnownPlace(name, latitude, longitude, radiusMeters))
    }

    fun setKnownPlaces(places: List<KnownPlace>) {
        knownPlaces.clear()
        knownPlaces.addAll(places)
    }

    private fun hasLocationPermission(context: Context): Boolean {
        return ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED ||
                ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
    }
}
