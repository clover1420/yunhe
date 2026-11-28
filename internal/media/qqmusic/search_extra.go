package qqmusic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// stringField reads a field QQ sends as either a string or a number.
func stringField(o object, keys ...string) string {
	for _, k := range keys {
		if s := text(o, k); s != "" {
			return s
		}
		var n json.Number
		if json.Unmarshal(o[k], &n) == nil && n.String() != "" {
			return n.String()
		}
	}
	return ""
}

// Hotkey is one of QQ's trending search words.
type Hotkey struct {
	ID          string
	Query       string
	Title       string
	Score       string
	Kind        int
	Type        int
	Source      int
	NeedTop     int
	Subpos      int
	SongType    int
	DirectID    int64
	JumpTab     string
	JumpURL     string
	CoverPicURL string
	PicURL      string
	Description string
}

// HotkeyList is QQ's trending search page.
type HotkeyList struct {
	HotkeyTime  string
	TrackListID string
	Items       []Hotkey
}

// SearchHotkeys returns QQ's trending search words.
func SearchHotkeys() (HotkeyList, error) {
	var data object
	if err := rpc("music.musicsearch.HotkeyService", "GetHotkeyForQQMusicMobile", map[string]any{"search_id": searchID()}, &data); err != nil {
		return HotkeyList{}, err
	}
	out := HotkeyList{
		HotkeyTime:  text(data, "hotkey_time"),
		TrackListID: stringField(data, "track_list_id"),
	}
	for _, entry := range objects(data, "vec_hotkey") {
		out.Items = append(out.Items, Hotkey{
			ID:          stringField(entry, "hotkey_id"),
			Query:       text(entry, "query"),
			Title:       text(entry, "title"),
			Score:       stringField(entry, "score"),
			Kind:        int(number(entry, "kind")),
			Type:        int(number(entry, "type")),
			Source:      int(number(entry, "source")),
			NeedTop:     int(number(entry, "need_top", "needtop")),
			Subpos:      int(number(entry, "subpos")),
			SongType:    int(number(entry, "song_type", "songtype")),
			DirectID:    number(entry, "direct_id", "directid"),
			JumpTab:     stringField(entry, "jump_tab", "jumptab"),
			JumpURL:     text(entry, "jump_url", "jumpurl"),
			CoverPicURL: text(entry, "cover_pic_url"),
			PicURL:      text(entry, "pic_url"),
			Description: text(entry, "description"),
		})
	}
	return out, nil
}

// QuickSearchItem is one candidate of a quick search category.
type QuickSearchItem struct {
	DocID  string
	ID     string
	MID    string
	Name   string
	Singer string
	Pic    string
	VID    string
}

// QuickSearchCategory is one category of candidates: songs, singers, albums
// or videos.
type QuickSearchCategory struct {
	Name  string
	Type  int
	Order int
	Count int64
	Items []QuickSearchItem
}

// QuickSearchResult is QQ's smartbox answer for a keyword.
type QuickSearchResult struct {
	Songs   QuickSearchCategory
	Singers QuickSearchCategory
	Albums  QuickSearchCategory
	Videos  QuickSearchCategory
}

// QuickSearch returns QQ's lightweight search candidates without spending a
// full search request. It uses the same smartbox endpoint as suggestions but
// also keeps the videos it returns.
func QuickSearch(keyword string) (QuickSearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return QuickSearchResult{}, nil
	}
	if len([]rune(keyword)) > 60 {
		return QuickSearchResult{}, fmt.Errorf("qqmusic: quick search exceeds 60 characters")
	}
	params := url.Values{
		"key": {keyword}, "format": {"json"}, "inCharset": {"utf-8"},
		"outCharset": {"utf-8"}, "platform": {"yqq"}, "needNewCode": {"0"},
	}
	req, err := http.NewRequest(http.MethodGet, "https://c.y.qq.com"+smartboxPath+"?"+params.Encode(), nil)
	if err != nil {
		return QuickSearchResult{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://y.qq.com/")
	b, err := readHTTP(req)
	if err != nil {
		return QuickSearchResult{}, err
	}
	var response struct {
		Code    *int   `json:"code"`
		Subcode int    `json:"subcode"`
		Data    object `json:"data"`
	}
	if err := json.Unmarshal(b, &response); err != nil {
		return QuickSearchResult{}, fmt.Errorf("qqmusic: malformed quick search response: %w", err)
	}
	if response.Code == nil {
		return QuickSearchResult{}, fmt.Errorf("qqmusic: missing quick search status")
	}
	if *response.Code != 0 || response.Subcode != 0 {
		code := *response.Code
		if code == 0 {
			code = response.Subcode
		}
		return QuickSearchResult{}, &ProtocolError{Code: code, Module: "cgi", Method: smartboxPath}
	}
	return QuickSearchResult{
		Songs:   quickSearchCategory(response.Data, "song"),
		Singers: quickSearchCategory(response.Data, "singer"),
		Albums:  quickSearchCategory(response.Data, "album"),
		Videos:  quickSearchCategory(response.Data, "mv"),
	}, nil
}

func quickSearchCategory(data object, key string) QuickSearchCategory {
	o := obj(data[key])
	category := QuickSearchCategory{
		Name:  text(o, "name"),
		Type:  int(number(o, "type")),
		Order: int(number(o, "order")),
		Count: number(o, "count"),
	}
	for _, item := range objects(o, "itemlist") {
		category.Items = append(category.Items, QuickSearchItem{
			DocID:  stringField(item, "docid"),
			ID:     stringField(item, "id"),
			MID:    text(item, "mid"),
			Name:   text(item, "name"),
			Singer: text(item, "singer"),
			Pic:    text(item, "pic"),
			VID:    text(item, "vid"),
		})
	}
	if category.Count == 0 {
		category.Count = int64(len(category.Items))
	}
	return category
}

// SearchMV is a video in a typed search, with the fields its result card shows.
type SearchMV struct {
	MV
	PlayCount   int64
	PublishDate string
	SingerID    int64
	SingerMID   string
	SingerName  string
}

// SearchByTypePage is one page of a typed search. QQ fills only the field for
// the requested search type.
type SearchByTypePage struct {
	SearchID    string
	Page        int
	PerPage     int
	NextPage    int
	EstimateSum int
	Total       int
	HasMore     bool
	Songs       []Track
	Artists     []ArtistProfile
	Albums      []AlbumProfile
	Playlists   []Playlist
	MVs         []SearchMV
	Users       []User
	AudioAlbums []AlbumProfile
}

// SearchByType searches one category. searchType is QQ's code: 0 songs, 1
// singers, 2 albums, 3 playlists, 4 MVs, 7 lyrics, 8 users, 10 ringtones, 15
// audio albums and 18 audio. page starts at 1; NextPage is -1 when no page
// remains.
func SearchByType(keyword string, searchType, num, page int) (SearchByTypePage, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return SearchByTypePage{}, nil
	}
	if len([]rune(keyword)) > 60 {
		return SearchByTypePage{}, fmt.Errorf("qqmusic: search exceeds 60 characters")
	}
	if num <= 0 {
		num = 10
	}
	num = min(num, 50)
	if page < 1 {
		page = 1
	}
	param := map[string]any{
		"searchid":      searchID(),
		"query":         keyword,
		"search_type":   searchType,
		"num_per_page":  num,
		"page_num":      page,
		"highlight":     1,
		"grp":           1,
		"selectors":     map[string]any{},
		"vec_selectors": []any{},
	}
	var data object
	if err := rpcWithComm("music.search.SearchCgiService", "DoSearchForQQMusicMobile", param, &data, searchAndroidComm(sessionSnapshot()), nil); err != nil {
		return SearchByTypePage{}, err
	}
	meta := obj(data["meta"])
	out := SearchByTypePage{
		SearchID:    text(meta, "searchid", "sid"),
		Page:        page,
		PerPage:     int(number(meta, "perpage")),
		NextPage:    int(number(meta, "nextpage")),
		EstimateSum: int(number(meta, "estimate_sum")),
		Total:       int(number(meta, "sum")),
	}
	out.HasMore = out.NextPage > 0 && out.NextPage != -1
	body := obj(data["body"])
	if body == nil {
		return SearchByTypePage{}, fmt.Errorf("qqmusic: malformed search response")
	}
	if songs := objects(body, "item_song"); len(songs) > 0 {
		out.Songs = searchTracks(songs)
	}
	if artists := objects(body, "singer", "item_singer"); len(artists) > 0 {
		out.Artists = decodeSearchEntries(artists, SearchArtists, 0, SearchAuto, 0, 0).Artists
	}
	if albums := objects(body, "item_album"); len(albums) > 0 {
		out.Albums = decodeSearchEntries(albums, SearchAlbums, 0, SearchAuto, 0, 0).Albums
	}
	for _, entry := range objects(body, "item_songlist") {
		pl := parseSquarePlaylist(entry)
		if pl.ID > 0 || pl.Name != "" {
			out.Playlists = append(out.Playlists, pl)
		}
	}
	for _, entry := range objects(body, "item_mv") {
		mv := searchVideo(entry)
		if mv.ID > 0 || mv.VID != "" || mv.Name != "" {
			out.MVs = append(out.MVs, mv)
		}
	}
	for _, entry := range objects(body, "item_user") {
		user := User{
			UserID:    number(entry, "uin", "userid", "userId", "id"),
			Nickname:  text(entry, "nick", "nickname", "name"),
			AvatarURL: text(entry, "avatar", "logo", "headpic", "pic"),
		}
		if user.UserID > 0 || user.Nickname != "" {
			out.Users = append(out.Users, user)
		}
	}
	if audios := objects(body, "item_audio"); len(audios) > 0 {
		out.AudioAlbums = decodeSearchEntries(audios, SearchAlbums, 0, SearchAuto, 0, 0).Albums
	}
	return out, nil
}

// searchAndroidComm builds the Android identity typed search is pinned to.
func searchAndroidComm(s Session) map[string]any {
	comm := map[string]any{"ct": 11, "cv": 14090008, "v": 14090008, "chid": "10003505", "tmeAppID": "qqmusic"}
	if s.valid() {
		comm["qq"], comm["authst"], comm["tmeLoginType"] = s.MusicID, s.MusicKey, s.LoginType
	}
	return comm
}

func searchVideo(o object) SearchMV {
	mv := SearchMV{
		MV: MV{
			ID:       number(o, "id", "sid", "mvid"),
			VID:      text(o, "vid"),
			Name:     text(o, "name", "mvname", "title"),
			Title:    text(o, "title", "title_main", "name"),
			Subtitle: text(o, "subtitle"),
			Cover:    text(o, "pic", "picurl", "cover_pic", "cover"),
			Duration: number(o, "duration"),
		},
		PlayCount:   number(o, "play_count", "playcnt"),
		PublishDate: text(o, "publish_date"),
		SingerID:    number(o, "singerid", "singer_id"),
		SingerMID:   text(o, "singermid", "singer_mid"),
		SingerName:  text(o, "singername", "singer_name"),
	}
	if mv.PublishDate == "" {
		if date := number(o, "pubdate", "publish_date"); date > 0 {
			mv.PublishDate = strconv.FormatInt(date, 10)
		}
	}
	for _, singer := range objects(o, "singer", "singers") {
		mv.Singers = append(mv.Singers, convertArtist(singer))
	}
	return mv
}
