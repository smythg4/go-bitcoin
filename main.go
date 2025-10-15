package main

import (
	"fmt"
	ellipticcurve "go-bitcoin/internal/elliptic_curve"
)

func main() {
	c := ellipticcurve.NewCurve(0, 7, 223)

	points := [][]int{[]int{192, 105}, []int{143, 98}, []int{47, 71}, []int{47, 71}, []int{47, 71}, []int{47, 71}}
	ns := []int{2, 2, 2, 4, 8, 21}

	if len(points) != len(ns) {
		fmt.Println("slice lengths aren't equal")
		return
	}

	for i, pair := range points {
		if len(pair) != 2 {
			fmt.Println("int pair isn't length 2")
			return
		}
		x := pair[0]
		y := pair[1]
		n := ns[i]

		pt, err := ellipticcurve.NewPoint(x, y, c)
		if err != nil {
			fmt.Printf("error generating point - %v", err)
		}
		res, err := pt.ScalarMul(n)
		if err != nil {
			fmt.Printf("error with scalar multiplication - %v", err)
		}
		fmt.Printf("%d * %v = %v\n", n, pt, res)
	}
}
