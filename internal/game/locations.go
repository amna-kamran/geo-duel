package game

import "math/rand"

type Location struct {
	Name      string
	Continent string
	Lat       float64
	Lng       float64
}

var Locations = []Location{
	{"Japan", "Asia", 36.2048, 138.2529},
	{"Brazil", "South America", -14.2350, -51.9253},
	{"Egypt", "Africa", 26.8206, 30.8025},
	{"Canada", "North America", 56.1304, -106.3468},
	{"Australia", "Oceania", -25.2744, 133.7751},
	{"France", "Europe", 46.2276, 2.2137},
	{"India", "Asia", 20.5937, 78.9629},
	{"South Africa", "Africa", -30.5595, 22.9375},
	{"Mexico", "North America", 23.6345, -102.5528},
	{"Norway", "Europe", 60.4720, 8.4689},
	{"Thailand", "Asia", 15.8700, 100.9925},
	{"Argentina", "South America", -38.4161, -63.6167},
	{"Kenya", "Africa", -0.0236, 37.9062},
	{"New Zealand", "Oceania", -40.9006, 174.8860},
	{"Iceland", "Europe", 64.9631, -19.0208},
	{"South Korea", "Asia", 35.9078, 127.7669},
	{"Peru", "South America", -9.1900, -75.0152},
	{"Morocco", "Africa", 31.7917, -7.0926},
	{"Vietnam", "Asia", 14.0583, 108.2772},
	{"Finland", "Europe", 61.9241, 25.7482},
}

func RandomLocation() Location {
	return Locations[rand.Intn(len(Locations))]
}
