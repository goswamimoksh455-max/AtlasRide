package matching

type candidate struct {
	driverID string
	distance float64
}

func scoreByDistance(dist float64) float64 {
	return dist //lower is better , so simple
}

func better(a, b candidate) bool {
	return a.distance < b.distance
}

// I will try to make it as per the Blogs, like having ETA,Fainess, Driver laod etc
