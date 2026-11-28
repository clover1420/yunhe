package qqmusic

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SongFavorites returns how many listeners favorited each song.
func SongFavorites(ids []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var data struct {
		Numbers map[string]int64 `json:"m_numbers"`
	}
	if err := rpc("music.musicasset.SongFavRead", "GetSongFansNumberById", map[string]any{"v_songId": ids}, &data); err != nil {
		return nil, err
	}
	for k, v := range data.Numbers {
		if id, err := strconv.ParseInt(k, 10, 64); err == nil {
			out[id] = v
		}
	}
	return out, nil
}

// SongLabel is a tag QQ attaches to a song, as its recommendation cards show.
type SongLabel struct {
	ID      int64  `json:"id"`
	Text    string `json:"tagTxt"`
	Icon    string `json:"tagIcon"`
	URL     string `json:"tagUrl"`
	Type    int    `json:"tagType"`
	Species int    `json:"species"`
}

func SongLabels(songID int64) ([]SongLabel, error) {
	var data struct {
		Labels []SongLabel `json:"labels"`
	}
	err := rpc("music.recommend.TrackRelationServer", "GetSongLabels", map[string]any{"songid": songID}, &data)
	return data.Labels, err
}

// OtherVersions returns the versions QQ links to a song: remixes, lives and
// covers, as the song page lists them.
func OtherVersions(songID int64) ([]Track, error) {
	return otherVersions(map[string]any{"songid": songID})
}

// OtherVersionsByMID is OtherVersions for a song known by its MID.
func OtherVersionsByMID(mid string) ([]Track, error) {
	return otherVersions(map[string]any{"songmid": mid})
}

func otherVersions(param map[string]any) ([]Track, error) {
	var data struct {
		Versions []object `json:"versionList"`
	}
	if err := rpc("music.musichallSong.OtherVersionServer", "GetOtherVersionSongs", param, &data); err != nil {
		return nil, err
	}
	return convertTracks(data.Versions), nil
}

// Producer is one of a song's producers, in a group by what they did.
type Producer struct {
	Type      int    `json:"Type"`
	Name      string `json:"Name"`
	Icon      string `json:"Icon"`
	Scheme    string `json:"Scheme"`
	SingerMID string `json:"SingerMid"`
	Follow    int    `json:"Follow"`
}

// ProducerGroup holds the producers of one role: the writers, the composer,
// the producer and on.
type ProducerGroup struct {
	Title     string     `json:"Title"`
	Type      int        `json:"Type"`
	Producers []Producer `json:"Producers"`
}

// ProducerInfo is a song's credits, and the summary QQ shows over them.
type ProducerInfo struct {
	Groups  []ProducerGroup
	Summary string
}

func SongProducers(songID int64) (*ProducerInfo, error) {
	var data struct {
		Groups  []ProducerGroup `json:"Lst"`
		Summary string          `json:"ReinforceMsg"`
	}
	if err := rpc("music.sociality.KolWorksTag", "SongProducer", map[string]any{"songid": songID}, &data); err != nil {
		return nil, err
	}
	return &ProducerInfo{Groups: data.Groups, Summary: data.Summary}, nil
}

// RelatedPlaylist is a playlist QQ recommends beside a song.
type RelatedPlaylist struct {
	ID          int64
	Name        string
	Cover       string
	Description string
	Songs       int64
	Plays       int64
	Creator     string
}

// RelatedPlaylists returns the playlists QQ recommends beside a song, and
// whether more follow: ask again with the ids returned as next.
func RelatedPlaylists(songID int64, cursor []int64) (list []RelatedPlaylist, hasMore bool, next []int64, err error) {
	if cursor == nil {
		cursor = []int64{}
	}
	param := map[string]any{"songid": songID, "vecPlaylist": cursor}
	var data object
	if err = rpc("music.recommend.TrackRelationServer", "GetRelatedPlaylist", param, &data); err != nil {
		return nil, false, nil, err
	}
	for _, group := range objects(data, "vecPlaylistNew") {
		for _, o := range objects(group, "playlists") {
			p := RelatedPlaylist{
				ID:          number(o, "id", "tid", "dissid"),
				Name:        text(o, "title", "dissname", "name", "dirName"),
				Cover:       text(o, "picurl", "cover", "logo", "picUrl"),
				Description: text(o, "desc", "description"),
				Songs:       number(o, "songnum", "songNum", "song_cnt"),
				Plays:       number(o, "listennum", "playCnt", "play_cnt"),
				Creator:     text(o, "creator"),
			}
			if p.ID > 0 {
				list = append(list, p)
				next = append(next, p.ID)
			}
		}
	}
	return list, number(data, "hasMore") != 0, next, nil
}

// MV is a music video, as QQ lists it beside a song or on an artist's page.
type MV struct {
	ID          int64
	VID         string
	Name        string
	Title       string
	Subtitle    string
	Cover       string
	PicFormat   string
	Plays       int64
	Duration    int64
	PublishDate int64
	Singers     []Artist
}

// RelatedMVs returns the music videos QQ links to a song. last is the id of
// the last video of the previous page, or 0 for the first.
func RelatedMVs(songID, last int64) ([]MV, bool, error) {
	param := map[string]any{"songid": strconv.FormatInt(songID, 10), "songtype": 1, "lastmvid": last}
	var data object
	if err := rpc("MvService.MvInfoProServer", "GetSongRelatedMv", param, &data); err != nil {
		return nil, false, err
	}
	var out []MV
	for _, o := range objects(data, "list") {
		mv := MV{
			ID:    number(o, "id", "sid", "mvid", "singerId"),
			VID:   text(o, "vid"),
			Name:  text(o, "name", "mvname", "title"),
			Title: text(o, "title", "title_main", "name"),
			Cover: text(o, "picurl"),
			Plays: number(o, "playcnt"),
		}
		for _, s := range objects(o, "singers") {
			mv.Singers = append(mv.Singers, convertArtist(s))
		}
		out = append(out, mv)
	}
	return out, number(data, "hasmore") != 0, nil
}

// SimilarSongGroup is one of QQ's cards of songs that sound like another.
type SimilarSongGroup struct {
	Title   string
	Content string
	Songs   []Track
}

func SimilarSongs(songID int64) ([]SimilarSongGroup, error) {
	var data object
	if err := rpc("music.recommend.TrackRelationServer", "GetSimilarSongs", map[string]any{"songid": songID}, &data); err != nil {
		return nil, err
	}
	var out []SimilarSongGroup
	for _, group := range objects(data, "vecSongNew") {
		g := SimilarSongGroup{
			Title:   text(group, "title_template", "titleTemplate"),
			Content: text(group, "title_content", "titleContent"),
		}
		for _, entry := range objects(group, "songs") {
			if track := obj(entry["track"]); track != nil {
				g.Songs = append(g.Songs, convertTrack(track))
			}
		}
		out = append(out, g)
	}
	return out, nil
}

// CDNNode is one of the CDN nodes QQ's dispatch names.
type CDNNode struct {
	CDN           string
	QUIC          int
	IPStack       int
	QUICHost      string
	PlaintextQUIC int
	EncryptQUIC   int
}

// CDNDispatch is where QQ serves audio from: sip holds the hosts to join a
// song's purl to, and keepalive the file to test them with.
type CDNDispatch struct {
	SIP           []string
	Nodes         []CDNNode
	KeepAliveFile string
	Expiration    int64
	RefreshTime   int64
	CacheTime     int64
}

func CDNDispatchInfo() (*CDNDispatch, error) {
	guid := make([]byte, 16)
	if _, err := rand.Read(guid); err != nil {
		return nil, err
	}
	param := map[string]any{"guid": hex.EncodeToString(guid), "uid": "0", "use_new_domain": 1, "use_ipv6": 1}
	var data struct {
		SIP           []string  `json:"sip"`
		Info          []CDNNode `json:"sipinfo"`
		KeepAliveFile string    `json:"keepalivefile"`
		Expiration    int64     `json:"expiration"`
		RefreshTime   int64     `json:"refreshTime"`
		CacheTime     int64     `json:"cacheTime"`
	}
	if err := rpc("music.audioCdnDispatch.cdnDispatch", "GetCdnDispatch", param, &data); err != nil {
		return nil, err
	}
	return &CDNDispatch{
		SIP:           data.SIP,
		Nodes:         data.Info,
		KeepAliveFile: data.KeepAliveFile,
		Expiration:    data.Expiration,
		RefreshTime:   data.RefreshTime,
		CacheTime:     data.CacheTime,
	}, nil
}

// QuerySongsByID looks songs up by their ids, all at once.
func QuerySongsByID(ids []int64) ([]Track, error) {
	types := make([]int, len(ids))
	stamps := make([]int, len(ids))
	return querySongs(map[string]any{"ctx": 0, "client": 1, "types": types, "modify_stamp": stamps, "ids": ids})
}

// QuerySongsByMID looks songs up by their MIDs, all at once.
func QuerySongsByMID(mids []string) ([]Track, error) {
	types := make([]int, len(mids))
	stamps := make([]int, len(mids))
	return querySongs(map[string]any{"ctx": 0, "client": 1, "types": types, "modify_stamp": stamps, "mids": mids})
}

func querySongs(param map[string]any) ([]Track, error) {
	var data struct {
		Tracks []object `json:"tracks"`
	}
	if err := rpc("music.trackInfo.UniformRuleCtrl", "CgiGetTrackInfo", param, &data); err != nil {
		return nil, err
	}
	return convertTracks(data.Tracks), nil
}

// Sheet is one score of a song, as its page lists them.
type Sheet struct {
	MID        string   `json:"mid"`
	Name       string   `json:"name"`
	Pics       []string `json:"pics"`
	Version    string   `json:"version"`
	Tonality   int      `json:"tonality"`
	ScoreType  int      `json:"scoreType"`
	TypeText   string   `json:"typeText"`
	Uploader   string   `json:"uploader"`
	Views      int64    `json:"views"`
	Author     string   `json:"author"`
	Composer   string   `json:"composer"`
	Lyricist   string   `json:"lyricist"`
	Singer     string   `json:"singer"`
	Performer  string   `json:"performer"`
	SongMID    string   `json:"songMid"`
	SubName    string   `json:"subName"`
	URL        string   `json:"url"`
	AlbumURL   string   `json:"albumUrl"`
	Instrument string   `json:"instrument"`
	CoverURL   string   `json:"coverUrl"`
	Difficulty string   `json:"difficulty"`
	File       string   `json:"file"`
}

const (
	SheetUploaded   = 0 // scores listeners uploaded
	SheetAI         = 1 // scores QQ's engine writes
	SheetChongChong = 2 // scores from ChongChong Piano
)

// Sheets returns the scores of a song of a kind: the ones listeners uploaded,
// the ones QQ's engine writes, or ChongChong Piano's. Ng a score of the kind
// leaves an empty list.
func Sheets(mid string, kind int) ([]Sheet, error) {
	comm := sheetComm(kind)
	module, method := "music.mir.SheetMusicSvr", "GetMoreSheetMusic"
	if kind == SheetChongChong {
		method = "GetChongChongSheetMusic"
	}
	scoreType, ttype := -1, kind
	if kind == SheetAI {
		scoreType = -473
	}
	if kind == SheetChongChong {
		ttype = 1
	}
	param := map[string]any{"songMid": mid, "begin": 0, "end": 100, "scoreType": scoreType, "ttype": ttype}
	var data struct {
		Result []struct {
			MID        string   `json:"scoreMID"`
			Name       string   `json:"scoreName"`
			Pics       []string `json:"picURLs"`
			Version    string   `json:"version"`
			Tonality   int      `json:"tonality"`
			ScoreType  int      `json:"scoreType"`
			TypeText   string   `json:"strScoreType"`
			Uploader   string   `json:"uploader"`
			Views      int64    `json:"viewFrequency"`
			Author     string   `json:"author"`
			Composer   string   `json:"composer"`
			Lyricist   string   `json:"lyricist"`
			Singer     string   `json:"singer"`
			Performer  string   `json:"performer"`
			SongMID    string   `json:"songMID"`
			SubName    string   `json:"subName"`
			URL        string   `json:"url"`
			AlbumURL   string   `json:"albumURL"`
			Instrument string   `json:"strInsType"`
			CoverURL   string   `json:"coverURL"`
			Difficulty string   `json:"difficulty"`
			File       string   `json:"sheetFile"`
		} `json:"result"`
	}
	var err error
	if kind == SheetChongChong {
		err = rpcSigned(module, method, param, comm, &data)
	} else {
		err = rpcWithComm(module, method, param, &data, comm, nil)
	}
	var protocol *ProtocolError
	if err != nil && !(errors.As(err, &protocol) && protocol.Code == 10007) {
		return nil, err
	}
	out := make([]Sheet, 0, len(data.Result))
	for _, s := range data.Result {
		out = append(out, Sheet{
			MID: s.MID, Name: s.Name, Pics: s.Pics, Version: s.Version,
			Tonality: s.Tonality, ScoreType: s.ScoreType, TypeText: s.TypeText,
			Uploader: s.Uploader, Views: s.Views, Author: s.Author,
			Composer: s.Composer, Lyricist: s.Lyricist, Singer: s.Singer,
			Performer: s.Performer, SongMID: s.SongMID, SubName: s.SubName,
			URL: s.URL, AlbumURL: s.AlbumURL, Instrument: s.Instrument,
			CoverURL: s.CoverURL, Difficulty: s.Difficulty, File: s.File,
		})
	}
	return out, nil
}

// SheetAvailability says what kinds of scores a song has.
type SheetAvailability struct {
	Guitar     bool `json:"hasGuitar"`
	More       bool `json:"hasMore"`
	Tab        bool `json:"hasLDY"`
	Staff      bool `json:"hasQRCX"`
	ChongChong bool `json:"hasChongChong"`
}

func HasSheet(mid string) (*SheetAvailability, error) {
	var data SheetAvailability
	if err := rpcWithComm("music.mir.SheetMusicSvr", "HasSheetMusic", map[string]any{"songMid": mid}, &data, sheetComm(SheetUploaded), nil); err != nil {
		return nil, err
	}
	return &data, nil
}

// sheetComm is the comm QQ's sheet service reads: h5's, which drops the
// client's own.
func sheetComm(kind int) map[string]any {
	comm := map[string]any{
		"g_tk": 5381, "uin": "", "format": "json", "inCharset": "utf-8",
		"outCharset": "utf-8", "notice": 0, "needNewCode": 1,
	}
	if kind == SheetChongChong {
		comm["platform"] = "h5"
	}
	return comm
}

// rpcSigned sends a request whose body is signed, as ChongChong's scores
// want, to musics.fcg.
func rpcSigned(module, method string, param any, comm map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"comm": comm, "req_0": map[string]any{"module": module, "method": method, "param": param}})
	if err != nil {
		return err
	}
	url := "https://u.y.qq.com/cgi-bin/musics.fcg?_=" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "&sign=" + zzcSign(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	b, err := readHTTP(req)
	if err != nil {
		return err
	}
	return decodeRPC(b, module, method, out)
}

// decodeRPC reads the reply of a signed request: the same envelope as
// musicu's.
func decodeRPC(b []byte, module, method string, out any) error {
	var body struct {
		Code *int `json:"code"`
		Req  *struct {
			Code    *int            `json:"code"`
			Subcode int             `json:"subcode"`
			Data    json.RawMessage `json:"data"`
		} `json:"req_0"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return fmt.Errorf("qqmusic: malformed RPC response: %w", err)
	}
	if body.Code == nil || body.Req == nil || body.Req.Code == nil {
		return fmt.Errorf("qqmusic: missing RPC status")
	}
	if *body.Code != 0 || *body.Req.Code != 0 || body.Req.Subcode != 0 {
		code := *body.Req.Code
		if code == 0 {
			code = body.Req.Subcode
		}
		return &ProtocolError{*body.Code, code, module, method}
	}
	if out == nil {
		return nil
	}
	if len(body.Req.Data) == 0 {
		return fmt.Errorf("qqmusic: missing RPC data")
	}
	return json.Unmarshal(body.Req.Data, out)
}

// zzcSign signs a request body as musics.fcg's zzc wants it.
func zzcSign(body []byte) string {
	sum := sha1.Sum(body)
	h := strings.ToUpper(hex.EncodeToString(sum[:]))
	part1 := string([]byte{h[23], h[14], h[6], h[36], h[16], h[7], h[19]})
	part2 := string([]byte{h[16], h[1], h[32], h[12], h[19], h[27], h[8], h[5]})
	scramble := []byte{89, 39, 179, 150, 218, 82, 58, 252, 177, 52, 186, 123, 120, 64, 242, 133, 143, 161, 121, 179}
	part3 := make([]byte, len(scramble))
	for i := range scramble {
		v, _ := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		part3[i] = scramble[i] ^ byte(v)
	}
	encoded := strings.NewReplacer(`\`, "", "/", "", "+", "", "=", "").Replace(base64.StdEncoding.EncodeToString(part3))
	return strings.ToLower("zzc" + part1 + encoded + part2)
}
