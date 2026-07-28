package com.andrometry.collectors.location

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.os.Looper
import androidx.core.content.ContextCompat
import com.andrometry.core.database.EventEntity
import com.andrometry.core.database.SequenceGenerator
import com.andrometry.core.database.TimelineSegmentEntity
import com.google.android.gms.location.*
import com.google.gson.Gson
import java.text.SimpleDateFormat
import java.util.*
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.abs

/**
 * Location collector using geofencing and passive location updates.
 * Follows the battery-efficient hierarchy from architecture.md:
 *
 * 1. Geofences for known places
 * 2. Passive location provider
 * 3. Significant location changes
 * 4. Low-frequency fused location updates
 * 5. High-frequency GPS only in debug mode
 *
 * Default cadence: Event-driven (geofence) + 15 min reconciliation
 * Produces: place_visit segments, location_context events, movement_context events
 */
@Singleton
class LocationCollector @Inject constructor(
    private val sequenceGenerator: SequenceGenerator
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

    // Default known places — user should configure these
    private val knownPlaces = mutableListOf(
        KnownPlace("home", 0.0, 0.0, 200f),   // Placeholder — set real coordinates
        KnownPlace("office", 0.0, 0.0, 150f),
        KnownPlace("gym", 0.0, 0.0, 100f),
        KnownPlace("cafe", 0.0, 0.0, 80f)
    )

    private var lastLocation: Location? = null
    private var lastPlace: String = "unknown"
    private var fusedLocationClient: FusedLocationProviderClient? = null
    private var geofencingClient: GeofencingClient? = null

    /**
     * Initializes geofences and starts passive location updates.
     */
    fun start(context: Context) {
        if (!hasLocationPermission(context)) return

        fusedLocationClient = LocationServices.getFusedLocationProviderClient(context)
        geofencingClient = LocationServices.getGeofencingClient(context)

        // Register geofences for known places
        registerGeofences(context)

        // Start passive location updates (low frequency)
        startPassiveLocationUpdates(context)
    }

    /**
     * Registers geofences for all known places.
     */
    private fun registerGeofences(context: Context) {
        val geofences = knownPlaces.map { place ->
            Geofence.Builder()
                .setRequestId(place.name)
                .setCircularRegion(place.latitude, place.longitude, place.radiusMeters)
                .setTransitionTypes(Geofence.GEOFENCE_TRANSITION_ENTER or Geofence.GEOFENCE_TRANSITION_EXIT)
                .setExpirationDuration(Geofence.NEVER_EXPIRE)
                .setLoiteringDelay(60000) // 1 minute dwell time
                .build()
        }

        val request = GeofencingRequest.Builder()
            .setInitialTrigger(GeofencingRequest.INITIAL_TRIGGER_ENTER)
            .addGeofences(geofences)
            .build()

        geofencingClient?.addGeofences(request, null)
    }

    /**
     * Starts passive location updates at low frequency (every 15 minutes).
     */
    private fun startPassiveLocationUpdates(context: Context) {
        if (!hasLocationPermission(context)) return

        val request = LocationRequest.Builder(Priority.PASSIVE_UPDATE, 15 * 60 * 1000L) // 15 min
            .setMinUpdateIntervalMillis(5 * 60 * 1000L) // 5 min minimum interval
            .build()

        fusedLocationClient?.requestLocationUpdates(request, object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                result.lastLocation?.let { location ->
                    processLocationUpdate(context, location)
                }
            }
        }, Looper.getMainLooper())
    }

    /**
     * Processes a location update and creates appropriate events/segments.
     */
    private fun processLocationUpdate(context: Context, location: Location) {
        val detectedPlace = detectPlace(location)
        val movement = classifyMovement(location)
        val placeChanged = detectedPlace != lastPlace

        // Create location context event
        val locationContext = LocationContext(
            place = detectedPlace,
            confidence = if (detectedPlace != "unknown") 0.9 else 0.5,
            movement = movement,
            changed = placeChanged
        )

        val payload = mapOf(
            "place" to locationContext.place,
            "confidence" to locationContext.confidence,
            "movement" to locationContext.movement,
            "changed" to locationContext.changed
        )

        // Store event
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

        val app = context.applicationContext as com.andrometry.AndrometryApp
        app.database.eventDao().insert(event)

        // If place changed, create place_visit segment
        if (placeChanged && lastPlace != "unknown") {
            // End previous place visit (approximate — actual start time should be tracked)
            val segment = TimelineSegmentEntity(
                segmentId = UUID.randomUUID().toString(),
                segmentType = "place_visit",
                layer = "place",
                startTime = now, // Should be tracked from when place was entered
                endTime = now,
                label = lastPlace,
                confidence = 0.88,
                validFrom = now,
                ingestedAt = now,
                payload = Gson().toJson(mapOf("confidence" to 0.88))
            )
            app.database.segmentDao().upsert(segment)
        }

        lastLocation = location
        lastPlace = detectedPlace
    }

    /**
     * Detects which known place the user is near.
     */
    private fun detectPlace(location: Location): String {
        for (place in knownPlaces) {
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

    /**
     * Classifies movement state from location updates.
     */
    private fun classifyMovement(location: Location): String {
        val last = lastLocation ?: return "unknown"

        val distance = FloatArray(1)
        Location.distanceBetween(
            last.latitude, last.longitude,
            location.latitude, location.longitude,
            distance
        )

        val timeDelta = (location.time - last.time) / 1000.0 // seconds
        if (timeDelta <= 0) return "still"

        val speed = distance[0] / timeDelta // m/s

        return when {
            speed < 0.5 -> "still"
            speed < 2.0 -> "walking"
            speed < 5.0 -> "running"
            else -> "vehicle"
        }
    }

    /**
     * Adds a custom known place.
     */
    fun addKnownPlace(name: String, latitude: Double, longitude: Double, radiusMeters: Float = 150f) {
        knownPlaces.add(KnownPlace(name, latitude, longitude, radiusMeters))
    }

    private fun hasLocationPermission(context: Context): Boolean {
        return ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED ||
                ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
    }
}
