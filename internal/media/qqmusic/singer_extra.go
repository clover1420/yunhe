package qqmusic

import "encoding/json"

// SingerBrief is a singer as QQ's lists show them.
type SingerBrief struct {
	ID         int64  `json:"id"`
	MID        string `json:"mid"`
	Name       string `json:"name"`
	PicURL     string `json:"picUrl"`
	Country    string `json:"country"`
	OtherName  string `json:"otherName"`
	Spell      string `json:"spell"`
	AreaID     int    `json:"areaId"`
	CountryID  int    `json:"countryId"`
	ConcernNum int64  `json:"concernNum"`
	Trend      int    `json:"trend"`
}

func parseSingerBrief(o object) SingerBrief {
	return SingerBrief{
		ID:         number(o, "singer_id", "singerId", "id"),
		MID:        text(o, "singer_mid", "singerMid", "mid"),
		Name:       text(o, "singer_name", "singerName", "name"),
		PicURL:     text(o, "singer_pic", "singerPic"),
		Country:    text(o, "country"),
		OtherName:  text(o, "other_name", "otherName"),
		Spell:      text(o, "spell"),
		AreaID:     int(number(o, "area_id", "areaId")),
		CountryID:  int(number(o, "country_id", "countryId")),
		ConcernNum: number(o, "concernNum"),
		Trend:      int(number(o, "trend")),
	}
}

func parseSingerBriefs(entries []object) []SingerBrief {
	out := make([]SingerBrief, 0, len(entries))
	for _, o := range entries {
		out = append(out, parseSingerBrief(o))
	}
	return out
}

// SingerTag is one choice of a singer list's filters.
type SingerTag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// SingerCategory is one area, sex and genre of QQ's singer list, with the
// tags its page offers.
type SingerCategory struct {
	Singers []SingerBrief
	Hot     []SingerBrief
	Tags    map[string][]SingerTag
}

// SingersByCategory returns the singers of an area, sex and genre, all at
// once. -100 is all of each.
func SingersByCategory(area, sex, genre int) (*SingerCategory, error) {
	param := map[string]any{"hastag": 0, "area": area, "sex": sex, "genre": genre}
	var data object
	if err := rpc("music.musichallSinger.SingerList", "GetSingerList", param, &data); err != nil {
		return nil, err
	}
	return &SingerCategory{
		Singers: parseSingerBriefs(objects(data, "singerlist")),
		Hot:     parseSingerBriefs(objects(data, "hotlist")),
		Tags:    singerTags(obj(data["tags"])),
	}, nil
}

// SingerIndexPage is a page of a singer list indexed by the first letter of
// their names.
type SingerIndexPage struct {
	Singers []SingerBrief
	Hot     []SingerBrief
	Tags    map[string][]SingerTag
	Total   int
}

// SingersByIndex returns a page of the singers of an area, sex, genre and
// first letter, page from 1. The index is 1 to 26 for A to Z, 27 for #, and
// -100 for all.
func SingersByIndex(area, sex, genre, index, page, num int) (*SingerIndexPage, error) {
	if page < 1 {
		page = 1
	}
	param := map[string]any{
		"area": area, "sex": sex, "genre": genre, "index": index,
		"sin": (page - 1) * num, "cur_page": page,
	}
	var data object
	if err := rpc("music.musichallSinger.SingerList", "GetSingerListIndex", param, &data); err != nil {
		return nil, err
	}
	return &SingerIndexPage{
		Singers: parseSingerBriefs(objects(data, "singerlist")),
		Hot:     parseSingerBriefs(objects(data, "hotlist")),
		Tags:    singerTags(obj(data["tags"])),
		Total:   int(number(data, "total")),
	}, nil
}

func singerTags(o object) map[string][]SingerTag {
	if o == nil {
		return nil
	}
	out := make(map[string][]SingerTag)
	for group, raw := range o {
		var entries []object
		if json.Unmarshal(raw, &entries) != nil {
			continue
		}
		tags := make([]SingerTag, 0, len(entries))
		for _, e := range entries {
			tags = append(tags, SingerTag{ID: number(e, "id"), Name: text(e, "name")})
		}
		out[group] = tags
	}
	return out
}

// SingerHomepage is the head of a singer's page: who they are, and what the
// page shows around them.
type SingerHomepage struct {
	Singer       SingerBrief
	Background   string
	Avatar       string
	EncryptedUin string
}

func SingerPageHeader(mid string) (*SingerHomepage, error) {
	param := map[string]any{"SingerMid": mid}
	var data object
	if err := rpc("music.UnifiedHomepage.UnifiedHomepageSrv", "GetHomepageHeader", param, &data); err != nil {
		return nil, err
	}
	info := obj(data["Info"])
	base := obj(info["BaseInfo"])
	return &SingerHomepage{
		Singer:       parseSingerBrief(obj(info["Singer"])),
		Background:   text(base, "BackgroundImage"),
		Avatar:       text(base, "Avatar"),
		EncryptedUin: text(base, "EncryptedUin"),
	}, nil
}

// SingerDetail is a singer's introduction, as QQ writes it.
type SingerDetail struct {
	SingerBrief
	Area        string   `json:"area"`
	Description string   `json:"desc"`
	Tag         string   `json:"tag"`
	Identity    string   `json:"identity"`
	Instrument  string   `json:"instrument"`
	Genre       string   `json:"genre"`
	ForeignName string   `json:"foreignName"`
	Birthday    string   `json:"birthday"`
	Wiki        string   `json:"wiki"`
	Pics        []string `json:"pics"`
}

// SingerDetails returns the introductions of singers, by their MIDs.
func SingerDetails(mids []string) ([]SingerDetail, error) {
	param := map[string]any{
		"singer_mids": mids, "group_singer": 1, "wiki_singer": 1,
		"ex_singer": 1, "pic": 1, "photos": 1,
	}
	var data object
	if err := rpc("music.musichallSinger.SingerInfoInter", "GetSingerDetail", param, &data); err != nil {
		return nil, err
	}
	var out []SingerDetail
	for _, o := range objects(data, "singer_list") {
		basic, ex := obj(o["basic_info"]), obj(o["ex_info"])
		d := SingerDetail{
			SingerBrief: parseSingerBrief(basic),
			Area:        text(ex, "area"),
			Description: text(ex, "desc"),
			Tag:         text(ex, "tag"),
			Identity:    text(ex, "identity"),
			Instrument:  text(ex, "instrument"),
			Genre:       text(ex, "genre"),
			ForeignName: text(ex, "foreign_name", "foreignName"),
			Birthday:    text(ex, "birthday"),
			Wiki:        text(o, "wiki"),
		}
		if pic := obj(o["pic"]); pic != nil {
			d.Pics = append(d.Pics, text(pic, "pic"))
		}
		for _, photo := range objects(o, "photos") {
			if big := text(photo, "big"); big != "" {
				d.Pics = append(d.Pics, big)
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// SimilarArtists returns the singers QQ recommends beside another.
func SimilarArtists(mid string, number int) ([]SingerBrief, error) {
	if number <= 0 {
		number = 10
	}
	param := map[string]any{"singerMid": mid, "number": number}
	var data object
	if err := rpc("music.SimilarSingerSvr", "GetSimilarSingerList", param, &data); err != nil {
		return nil, err
	}
	return parseSingerBriefs(objects(data, "singerlist")), nil
}

// AlbumBrief is an album as a singer's page lists them.
type AlbumBrief struct {
	ID          int64    `json:"id"`
	MID         string   `json:"mid"`
	Name        string   `json:"name"`
	Subtitle    string   `json:"subtitle"`
	PublishDate string   `json:"publishDate"`
	Total       int      `json:"total"`
	TypeText    string   `json:"typeText"`
	SingerName  string   `json:"singerName"`
	Tags        []string `json:"tags"`
}

func parseAlbumBrief(o object) AlbumBrief {
	return AlbumBrief{
		ID:          number(o, "albumID", "albumId", "id"),
		MID:         text(o, "albumMid", "albumMID", "albummid", "mid"),
		Name:        text(o, "albumName", "name"),
		Subtitle:    text(o, "albumTranName", "subtitle"),
		PublishDate: text(o, "publishDate", "publish_date"),
		Total:       int(number(o, "totalNum", "total")),
		TypeText:    text(o, "albumType"),
		SingerName:  text(o, "singerName"),
		Tags:        stringsOf(o["tags"]),
	}
}

func parseAlbumBriefs(entries []object) []AlbumBrief {
	out := make([]AlbumBrief, 0, len(entries))
	for _, o := range entries {
		out = append(out, parseAlbumBrief(o))
	}
	return out
}

// SingerSongs returns a page of a singer's songs, hottest first unless
// order is 0 for latest, and how many they have.
func SingerSongs(mid string, order, page, num int) ([]Track, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"singerMid": mid, "order": order, "number": num, "begin": (page - 1) * num}
	var data object
	if err := rpc("musichall.song_list_server", "GetSingerSongList", param, &data); err != nil {
		return nil, 0, err
	}
	var entries []object
	for _, entry := range objects(data, "songList") {
		entries = append(entries, obj(entry["songInfo"]))
	}
	return convertTracks(entries), int(number(data, "totalNum")), nil
}

// SingerAlbums returns a page of a singer's albums, hottest first unless
// order is 0 for latest, and how many they have. filterType -1 takes all.
func SingerAlbums(mid string, order, page, num, filterType int) ([]AlbumBrief, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"singerMid": mid, "order": order, "number": num, "begin": (page - 1) * num}
	if filterType >= 0 {
		param["types"] = []int{filterType}
	}
	var data object
	if err := rpc("music.musichallAlbum.AlbumListServer", "GetAlbumList", param, &data); err != nil {
		return nil, 0, err
	}
	return parseAlbumBriefs(objects(data, "albumList")), int(number(data, "total")), nil
}

// SingerMVTags returns the tags a singer's videos are sorted under.
func SingerMVTags(mid string) ([]SingerTag, error) {
	var data object
	if err := rpc("MvService.MvInfoProServer", "GetSingerMvTag", map[string]any{"singermid": mid}, &data); err != nil {
		return nil, err
	}
	var out []SingerTag
	for _, o := range objects(data, "list") {
		out = append(out, SingerTag{ID: number(o, "id"), Name: text(o, "name")})
	}
	return out, nil
}

// SingerMVs returns a page of a singer's videos, hottest first unless order
// is 0 for latest, and how many they have. tagID -1 takes all.
func SingerMVs(mid string, order, page, num, tagID int) ([]MV, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"singermid": mid, "order": order, "count": num, "start": (page - 1) * num}
	if tagID >= 0 {
		param["tagid"] = tagID
	}
	var data object
	if err := rpc("MvService.MvInfoProServer", "GetSingerMvList", param, &data); err != nil {
		return nil, 0, err
	}
	return parseMVs(objects(data, "list")), int(number(data, "total")), nil
}

// SingerTabDetail is one tab of a singer's page: the songs they wrote, the
// albums they made, or the videos they shot.
type SingerTabDetail struct {
	TabID   string
	HasMore bool
	Songs   []Track
	Albums  []AlbumBrief
	Videos  []MV
}

// SingerTab returns a tab of a singer's page. tab is QQ's id for it: "song_sing",
// "song_composing", "album", "video" and on. page starts at 1.
func SingerTab(mid, tab string, page, num, order int) (*SingerTabDetail, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{
		"SingerMid": mid, "IsQueryTabDetail": 1, "TabID": tab,
		"PageNum": page - 1, "PageSize": num, "Order": order,
	}
	var data object
	if err := rpc("music.UnifiedHomepage.UnifiedHomepageSrv", "GetHomepageTabDetail", param, &data); err != nil {
		return nil, err
	}
	out := &SingerTabDetail{
		TabID:   text(data, "TabID"),
		HasMore: number(data, "HasMore") != 0,
	}
	for _, o := range objects(obj(data["SongTab"]), "List") {
		out.Songs = append(out.Songs, convertTrack(o))
	}
	out.Albums = parseAlbumBriefs(objects(obj(data["AlbumTab"]), "AlbumList"))
	out.Videos = parseMVs(objects(obj(data["VideoTab"]), "VideoList"))
	return out, nil
}

// parseMVs decodes QQ's video lists, shared by a song's, a singer's and the
// site's.
func parseMVs(entries []object) []MV {
	out := make([]MV, 0, len(entries))
	for _, o := range entries {
		mv := MV{
			ID:          number(o, "id", "sid", "mvid", "singerId"),
			VID:         text(o, "vid"),
			Name:        text(o, "name", "mvname", "title"),
			Title:       text(o, "title", "title_main", "name"),
			Subtitle:    text(o, "subtitle"),
			Cover:       text(o, "picurl", "cover_pic"),
			PicFormat:   text(o, "picformat"),
			Plays:       number(o, "playcnt"),
			Duration:    number(o, "duration"),
			PublishDate: number(o, "pubdate"),
		}
		for _, s := range objects(o, "singers") {
			mv.Singers = append(mv.Singers, convertArtist(s))
		}
		out = append(out, mv)
	}
	return out
}

func stringsOf(raw []byte) []string {
	var out []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}
