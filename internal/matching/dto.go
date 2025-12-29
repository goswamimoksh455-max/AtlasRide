package matching

//data transfer object
type MatchRequest struct {
	RiderID string
	Lat     float64
	Lng     float64
	K       int     // H3 k-ring Radious - 3*K(K+1)+1 Unambiguis Neighbors //so using H3-V4 successor of the K-ring called GridDisk (may fail at Pentagones)
	MaxDist float64 //meters
}

type MatchResult struct {
	RiderID  string
	DriverID string
	Distance float64
}
