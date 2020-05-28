package main

// Tree is a noble growing thing with relative and absolute (on a yard grid) locations.
type Tree struct {
	ID               int32  `json:"id"`
	Species          string `json:"species"`
	XCoord           int    `json:"x"`
	YCoord           int    `json:"y"`
	RelativeLocation string `json:"relativeLocation,omitempty"`
	Fallen           bool   `json:"fallen"`
}
