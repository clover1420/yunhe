package qqmusic

// AlbumInfo is an album's page: what it is, who made it, and what it says
// about itself.
type AlbumInfo struct {
	AlbumProfile
	Description string
	Language    int
	Genre       int
	AlbumType   int
	WikiURL     string
	Company     string
}

// AlbumInfoByMID returns an album's page by its MID, with what its release
// notice says and the company that put it out.
func AlbumInfoByMID(mid string) (*AlbumInfo, error) {
	var data object
	if err := rpc("music.musichallAlbum.AlbumInfoServer", "GetAlbumDetail", map[string]any{"albumMId": mid}, &data); err != nil {
		return nil, err
	}
	basic := obj(data["basicInfo"])
	a := AlbumInfo{
		AlbumProfile: AlbumProfile{Album: convertAlbum(basic)},
		Description:  text(basic, "desc"),
		Language:     int(number(basic, "language")),
		Genre:        int(number(basic, "genre")),
		AlbumType:    int(number(basic, "albumType")),
		WikiURL:      text(basic, "wikiurl"),
		Company:      text(obj(data["company"]), "name"),
	}
	for _, s := range objects(obj(data["singer"]), "singerList") {
		a.Artists = append(a.Artists, convertArtist(s))
	}
	return &a, nil
}

// AlbumSongs returns a page of an album's songs, and how many there are.
func AlbumSongs(mid string, page, num int) ([]Track, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"begin": (page - 1) * num, "num": num, "albumMid": mid}
	var data object
	if err := rpc("music.musichallAlbum.AlbumSongList", "GetAlbumSongList", param, &data); err != nil {
		return nil, 0, err
	}
	var entries []object
	for _, e := range objects(data, "songList") {
		entries = append(entries, obj(e["songInfo"]))
	}
	return convertTracks(entries), int(number(data, "totalNum")), nil
}

// NewAlbums returns a page of the albums QQ has just released, and how many
// there are. area 1 is the mainland, 2 Hong Kong and Taiwan, 3 Europe and
// America, 4 Korea, 5 Japan, 6 elsewhere.
func NewAlbums(area, page, num int) ([]AlbumProfile, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 20
	}
	param := map[string]any{"area": area, "num": num, "start": (page - 1) * num}
	var data object
	if err := rpc("newalbum.NewAlbumServer", "get_new_album_info", param, &data); err != nil {
		return nil, 0, err
	}
	var out []AlbumProfile
	for _, o := range objects(data, "albums") {
		a := AlbumProfile{Album: convertAlbum(o)}
		for _, s := range objects(o, "singers") {
			a.Artists = append(a.Artists, convertArtist(s))
		}
		out = append(out, a)
	}
	return out, int(number(data, "total")), nil
}
