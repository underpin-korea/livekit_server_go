package selector

import "errors"

var (
	ErrNoAvailableNodes           = errors.New("could not find any available nodes")
	ErrCurrentRegionNotSet        = errors.New("current region cannot be blank")
	ErrCurrentRegionUnknownLatLon = errors.New("unknown lat and lon for the current region")
)
