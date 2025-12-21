package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/goswamimoksh455-max/projects/AtlasRide/qtree"
)

func main() {
	boundary := &qtree.Rectangle{
		X:  300,
		Y:  300,
		HW: 300,
		HH: 300,
	}
	qt := qtree.NewQtree(boundary, 4)

	// for i := 0; i < 500; i++ {
	// 	p := qtree.Point{
	// 		X: rand.Float64()*600 - 300,
	// 		Y: rand.Float64()*600 - 300,
	// 	}
	// 	if qt.Insert(p) {
	// 		//fmt.Println("ok")
	// 	}

	// }

	if qt.Insert(qtree.Point{X: 100, Y: 100}) {
		fmt.Println("ok")
	}
	if qt.Insert(qtree.Point{X: 110, Y: 110}) {
		fmt.Println("ok")
	}
	if qt.Insert(qtree.Point{X: 120, Y: 120}) {
		fmt.Println("ok")
	}
	if qt.Insert(qtree.Point{X: 130, Y: 130}) {
		fmt.Println("ok")
	}
	if qt.Insert(qtree.Point{X: 140, Y: 140}) {
		fmt.Println("ok")
	}

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(qt)
	})

	http.HandleFunc("/in", func(w http.ResponseWriter, r *http.Request) {
		x, _ := strconv.ParseFloat(r.URL.Query().Get("x"), 64)
		y, _ := strconv.ParseFloat(r.URL.Query().Get("y"), 64)
		qt.Insert(qtree.Point{X: x, Y: y})

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(qt)
	})

	fmt.Println("server starting at http://localhost:8080/data")
	http.ListenAndServe(":8080", nil)
}
