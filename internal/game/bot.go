package game

import (
	"math"
	"math/rand"
)

const botName = "GeoBot"

// destinationPoint returns the point reached by travelling distanceKm from
// (lat, lng) along the given bearing (degrees, 0 = north), using the
// standard spherical great-circle formula.
func destinationPoint(lat, lng, bearingDeg, distanceKm float64) (float64, float64) {
	angularDist := distanceKm / earthRadiusKm
	bearing := bearingDeg * math.Pi / 180
	lat1 := lat * math.Pi / 180
	lng1 := lng * math.Pi / 180

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDist) +
		math.Cos(lat1)*math.Sin(angularDist)*math.Cos(bearing))
	lng2 := lng1 + math.Atan2(
		math.Sin(bearing)*math.Sin(angularDist)*math.Cos(lat1),
		math.Cos(angularDist)-math.Sin(lat1)*math.Sin(lat2),
	)

	return lat2 * 180 / math.Pi, lng2 * 180 / math.Pi
}

// botGuess produces a guess near loc, deliberately imperfect: it targets a
// score somewhat below humanAvgScore so the bot plays like a slightly
// weaker opponent rather than a perfect one, with randomness so it doesn't
// always miss by the same amount or in the same direction.
func botGuess(loc Location, humanAvgScore float64) LatLng {
	target := humanAvgScore * (0.55 + rand.Float64()*0.25) // aim for ~55-80% of their average
	target = math.Max(200, math.Min(target, 4500))

	targetDistKm := -2000 * math.Log(target/5000)
	// add +/-40% jitter so the bot's error radius varies round to round
	distKm := targetDistKm * (0.6 + rand.Float64()*0.8)
	if distKm < 5 {
		distKm = 5
	}

	bearing := rand.Float64() * 360
	lat, lng := destinationPoint(loc.Lat, loc.Lng, bearing, distKm)

	lat = math.Max(-85, math.Min(85, lat))
	for lng > 180 {
		lng -= 360
	}
	for lng < -180 {
		lng += 360
	}

	return LatLng{Lat: lat, Lng: lng}
}
