package app

import "lilt/internal/media/art"

// The app reads covers through these names while its views still live in
// this package; feature packages import lilt/internal/media/art directly.
type artwork = art.Artwork
type artCache = art.Cache
type artNeed = art.Needs

const artCover = art.Cover

var (
	newArtCache = art.New

	fetchImage    = art.FetchImage
	decodeArt     = art.Decode
	dominantColor = art.DominantColor
	blurImage     = art.BlurImage
)
