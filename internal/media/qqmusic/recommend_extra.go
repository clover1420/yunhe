package qqmusic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FeedCardKind names the entity a home feed card opens.
type FeedCardKind string

const (
	FeedCardSong     FeedCardKind = "song"
	FeedCardPlaylist FeedCardKind = "playlist"
	FeedCardAlbum    FeedCardKind = "album"
	FeedCardVideo    FeedCardKind = "video"
	FeedCardOther    FeedCardKind = "other"
)

// FeedCard is one home feed card: the type codes QQ uses to open it, its
// title and cover, and the entity it shows when the feed includes one.
type FeedCard struct {
	ID       string
	Type     int
	Subtype  int
	JumpType int
	Op       string
	Kind     FeedCardKind
	Title    string
	Subtitle string
	Cover    string
	Count    int64
	Track    *Track
	Playlist *Playlist
	Album    *Album
	Video    *MV
}

// HomeFeedShelf is one titled group of home feed cards.
type HomeFeedShelf struct {
	ID            int64
	TitleTemplate string
	TitleContent  string
	Title         string
	Cards         []FeedCard
}

// HomeFeedPage is one page of QQ Music's home feed.
type HomeFeedPage struct {
	Prompt   string
	LoadMark int64
	DNum     int64
	Page     int
	Shelves  []HomeFeedShelf
}

// HomeFeed reads one page of the home feed. page starts at 1; the first page
// uses direction 0, and later pages use direction 1 with the shelf count
// already loaded in sNum and their ids in vCache.
func HomeFeed(page, direction, sNum int, vCache []string) (HomeFeedPage, error) {
	if page < 1 {
		page = 1
	}
	if direction != 1 {
		direction = 0
	}
	if sNum < 0 {
		sNum = 0
	}
	if vCache == nil {
		vCache = []string{}
	}
	const module, method = "music.recommend.RecommendFeed", "get_recommend_feed"
	param := map[string]any{"direction": direction, "page": page, "s_num": sNum, "v_cache": vCache}
	var data object
	if err := rpcWithComm(module, method, param, &data, songFeedComm(sessionSnapshot()), nil); err != nil {
		return HomeFeedPage{}, err
	}
	if _, ok := data["retcode"]; !ok {
		return HomeFeedPage{}, fmt.Errorf("qqmusic: missing home feed status")
	}
	if code := number(data, "retcode"); code != 0 {
		return HomeFeedPage{}, &ProtocolError{0, int(code), module, method}
	}
	raw, ok := data["v_shelf"]
	if !ok {
		return HomeFeedPage{}, fmt.Errorf("qqmusic: missing home feed shelves")
	}
	var shelves []object
	if err := json.Unmarshal(raw, &shelves); err != nil {
		return HomeFeedPage{}, fmt.Errorf("qqmusic: malformed home feed shelves: %w", err)
	}
	feed := HomeFeedPage{
		Prompt:   text(data, "prompt"),
		LoadMark: number(data, "load_mark"),
		DNum:     number(data, "d_num"),
		Page:     page,
	}
	for _, shelf := range shelves {
		s := HomeFeedShelf{
			ID:            number(shelf, "id"),
			TitleTemplate: text(shelf, "title_template"),
			TitleContent:  text(shelf, "title_content"),
		}
		s.Title = feedTitle(s.TitleTemplate, s.TitleContent)
		for _, niche := range objects(shelf, "v_niche") {
			for _, card := range objects(niche, "v_card") {
				s.Cards = append(s.Cards, parseFeedCard(card))
			}
		}
		feed.Shelves = append(feed.Shelves, s)
	}
	return feed, nil
}

// feedTitle resolves the {String} placeholder QQ puts in shelf titles.
func feedTitle(template, content string) string {
	title := template
	if title == "" {
		title = content
	} else {
		title = strings.ReplaceAll(title, "{String}", content)
	}
	return strings.TrimSpace(strings.ReplaceAll(title, "\u2062", ""))
}

func parseFeedCard(o object) FeedCard {
	card := FeedCard{
		ID:       stringField(o, "id"),
		Type:     int(number(o, "type")),
		Subtype:  int(number(o, "subtype")),
		JumpType: int(number(o, "jumptype", "jump_type")),
		Op:       text(o, "op", "op_type"),
		Title:    text(o, "title", "name", "title_content"),
		Subtitle: text(o, "subtitle"),
		Cover:    text(o, "cover", "cover_url", "picurl", "pic"),
		Count:    number(o, "cnt", "count"),
	}
	switch {
	case feedNested(o, "song", "track") != nil:
		card.Kind, card.Track = FeedCardSong, feedTrack(feedNested(o, "song", "track"))
	case card.Type == 200:
		card.Kind, card.Track = FeedCardSong, feedTrack(o)
	case card.Type == 300 || card.Type == 301:
		card.Kind, card.Video = FeedCardVideo, feedVideo(o)
	case card.Type == 500 && card.Subtype == 0 && card.JumpType == 10014:
		card.Kind, card.Playlist = FeedCardPlaylist, feedPlaylist(o, card)
	case feedNested(o, "album") != nil:
		card.Kind, card.Album = FeedCardAlbum, feedAlbum(feedNested(o, "album"))
	case text(o, "albummid", "albumMid", "album_mid") != "" || number(o, "albumid", "albumID") > 0:
		card.Kind, card.Album = FeedCardAlbum, feedAlbum(o)
	default:
		card.Kind = FeedCardOther
	}
	return card
}

func feedNested(o object, keys ...string) object {
	for _, k := range keys {
		if nested := obj(o[k]); len(nested) > 0 {
			return nested
		}
	}
	return nil
}

func feedTrack(o object) *Track {
	if o == nil {
		return nil
	}
	t := convertTrack(o)
	if t.Name == "" {
		t.Name = text(o, "title", "songname", "songName")
	}
	if t.ID <= 0 && t.MID == "" && t.Name == "" {
		return nil
	}
	return &t
}

func feedPlaylist(o object, card FeedCard) *Playlist {
	if nested := feedNested(o, "playlist"); nested != nil {
		o = nested
	}
	pl := Playlist{
		ID:          number(o, "tid", "dissid", "id"),
		Name:        text(o, "title", "name", "dissname"),
		Description: text(o, "desc", "description"),
		TrackCount:  int(number(o, "song_cnt", "songnum", "songNum")),
		PlayCount:   float64(number(o, "play_cnt", "listennum", "listen_num")),
	}
	if cover := obj(o["cover"]); len(cover) > 0 {
		pl.CoverURL = text(cover, "default_url", "big_url", "medium_url", "small_url")
	}
	if pl.CoverURL == "" {
		pl.CoverURL = text(o, "cover_url", "imgurl", "picurl", "logo", "cover")
	}
	if creator := obj(o["creator"]); len(creator) > 0 {
		pl.Creator.Nickname = text(creator, "nick", "nickname", "name")
	}
	if pl.ID <= 0 {
		if id, err := strconv.ParseInt(card.ID, 10, 64); err == nil && id > 0 {
			pl.ID = id
		}
	}
	if pl.Name == "" {
		pl.Name = card.Title
	}
	if pl.CoverURL == "" {
		pl.CoverURL = card.Cover
	}
	if pl.PlayCount == 0 {
		pl.PlayCount = float64(card.Count)
	}
	if pl.ID <= 0 && pl.Name == "" {
		return nil
	}
	return &pl
}

func feedAlbum(o object) *Album {
	if o == nil {
		return nil
	}
	a := convertAlbum(o)
	if a.Name == "" {
		a.Name = text(o, "title", "name")
	}
	if a.PicURL == "" {
		a.PicURL = text(o, "pic", "cover", "cover_url")
	}
	if a.ID <= 0 && a.MID == "" && a.Name == "" {
		return nil
	}
	return &a
}

func feedVideo(o object) *MV {
	if o == nil {
		return nil
	}
	videos := parseMVs([]object{o})
	if len(videos) == 0 {
		return nil
	}
	mv := videos[0]
	if mv.Cover == "" {
		mv.Cover = text(o, "pic", "cover", "cover_url")
	}
	if mv.ID <= 0 && mv.VID == "" && mv.Name == "" {
		return nil
	}
	return &mv
}

// GuessRecommend returns the songs QQ recommends as "guess you like". QQ
// needs a signed-in identity for this feed.
func GuessRecommend() ([]Track, error) {
	param := map[string]any{"id": 99, "num": 5, "from": 0, "scene": 0, "song_ids": []any{}}
	var data object
	if err := rpc("music.radioProxy.MbTrackRadioSvr", "get_radio_track", param, &data); err != nil {
		return nil, err
	}
	return convertTracks(objects(data, "tracks")), nil
}

// RadarRecommendPage is one page of the radar recommendation stream.
type RadarRecommendPage struct {
	Songs            []Track
	RecommendSongIDs []int64
	BaseSongIDs      []int64
	HasMore          bool
	Toast            string
	Timestamp        int64
	VideoCards       json.RawMessage
}

// RadarRecommend reads the radar recommendation stream, page from 1.
func RadarRecommend(page int) (RadarRecommendPage, error) {
	if page < 1 {
		page = 1
	}
	param := map[string]any{"Page": page, "ReqType": 0, "FavSongs": []any{}, "EntranceSongs": []any{}}
	var data object
	if err := rpc("music.recommend.TrackRelationServer", "GetRadarSong", param, &data); err != nil {
		return RadarRecommendPage{}, err
	}
	out := RadarRecommendPage{
		RecommendSongIDs: int64Values(data["RecommendSongIds"]),
		BaseSongIDs:      int64Values(data["BaseSongIds"]),
		HasMore:          boolField(data, "HasMore"),
		Toast:            text(data, "Toast"),
		Timestamp:        number(data, "TimeStamp"),
		VideoCards:       data["VideoCards"],
	}
	for _, entry := range objects(data, "VecSongs") {
		track := obj(entry["Track"])
		if len(track) == 0 {
			track = entry
		}
		if t := feedTrack(track); t != nil {
			out.Songs = append(out.Songs, *t)
		}
	}
	return out, nil
}

func int64Values(raw json.RawMessage) []int64 {
	var out []int64
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// NewSongLanguage is one region or language channel of the new song feed.
type NewSongLanguage struct {
	Lan  string
	Type int
}

// NewSongTag is one tag of the new song feed.
type NewSongTag struct {
	ID       int64
	TagID    int64
	Tag      string
	Link     string
	FromType int
}

// RecommendNewSongPage is the new song feed for one region or language.
type RecommendNewSongPage struct {
	LanList []NewSongLanguage
	Lan     string
	Songs   []Track
	RetMsg  string
	Type    int
	Tags    []NewSongTag
}

// RecommendNewSongs returns QQ's new song feed. areaType is 1 mainland, 2
// Europe and America, 3 Japan, 4 Korea, 5 latest and 6 Hong Kong and Taiwan.
func RecommendNewSongs(areaType int) (RecommendNewSongPage, error) {
	if areaType <= 0 {
		areaType = 5
	}
	var data object
	if err := rpc("newsong.NewSongServer", "get_new_song_info", map[string]any{"type": areaType}, &data); err != nil {
		return RecommendNewSongPage{}, err
	}
	out := RecommendNewSongPage{
		Lan:    text(data, "lan"),
		Songs:  convertTracks(objects(data, "songlist")),
		RetMsg: text(data, "ret_msg"),
		Type:   int(number(data, "type")),
	}
	for _, entry := range objects(data, "lanlist") {
		out.LanList = append(out.LanList, NewSongLanguage{Lan: text(entry, "lan", "name"), Type: int(number(entry, "type"))})
	}
	for _, entry := range objects(data, "songTagInfoList") {
		out.Tags = append(out.Tags, NewSongTag{
			ID:       number(entry, "id"),
			TagID:    number(entry, "tagid", "tag_id"),
			Tag:      text(entry, "tag"),
			Link:     text(entry, "link"),
			FromType: int(number(entry, "from_type", "fromType")),
		})
	}
	return out, nil
}

// RecommendSonglistPage is one cursor page of the recommended playlists.
type RecommendSonglistPage struct {
	Songlists []Playlist
	HasMore   bool
	From      int64
	Msg       string
}

// RecommendSonglists reads the playlist square's recommended lists. page
// starts at 1, num lists per page, and the reply's From is the next cursor.
func RecommendSonglists(page, num int) (RecommendSonglistPage, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 25
	}
	num = min(num, 200)
	param := map[string]any{"From": num * (page - 1), "Size": num}
	var data object
	if err := rpcSquare(context.Background(), "music.playlist.PlaylistSquare", "GetRecommendFeed", param, &data); err != nil {
		return RecommendSonglistPage{}, err
	}
	out := RecommendSonglistPage{
		HasMore: boolField(data, "HasMore"),
		From:    number(data, "FromLimit"),
		Msg:     text(data, "Msg"),
	}
	for _, entry := range objects(data, "List") {
		basic := obj(obj(entry["Playlist"])["basic"])
		if len(basic) == 0 {
			basic = obj(entry["basic"])
		}
		if len(basic) == 0 {
			continue
		}
		pl := Playlist{
			ID:          number(basic, "tid", "dissid", "id"),
			Name:        text(basic, "title", "name"),
			Description: text(basic, "desc", "description"),
			TrackCount:  int(number(basic, "song_cnt", "songnum", "songNum")),
			PlayCount:   float64(number(basic, "play_cnt", "listennum", "listen_num")),
		}
		if cover := obj(basic["cover"]); len(cover) > 0 {
			pl.CoverURL = text(cover, "default_url", "big_url", "medium_url", "small_url")
		}
		if pl.CoverURL == "" {
			pl.CoverURL = text(basic, "cover_url", "imgurl", "picurl", "logo")
		}
		if creator := obj(basic["creator"]); len(creator) > 0 {
			pl.Creator.Nickname = text(creator, "nick", "nickname", "name")
		}
		if pl.ID > 0 || pl.Name != "" {
			out.Songlists = append(out.Songlists, pl)
		}
	}
	out.HasMore = out.HasMore && len(out.Songlists) > 0
	return out, nil
}
