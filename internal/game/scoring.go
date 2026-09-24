package game

import "math"

const earthRadiusKm = 6371.0

func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// Score rewards close guesses on an exponential curve, maxing at 5000.
func Score(distanceKm float64) int {
	s := 5000 * math.Exp(-distanceKm/2000)
	return int(math.Round(s))
}
