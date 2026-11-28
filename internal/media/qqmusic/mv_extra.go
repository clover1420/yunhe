package qqmusic

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

// MVDetail is a music video as its page shows it.
type MVDetail struct {
	MV
	Description  string
	Favorite     bool
	Uploader     string
	RelatedSongs []int64
}

// MVDetails returns the videos of the vids given, by vid.
func MVDetails(vids []string) (map[string]MVDetail, error) {
	required := []string{
		"vid", "type", "sid", "cover_pic", "duration", "singers", "video_switch",
		"msg", "name", "desc", "playcnt", "pubdate", "isfav", "gmid",
		"uploader_headurl", "uploader_nick", "uploader_encuin", "uploader_uin",
		"uploader_hasfollow", "uploader_follower_num", "related_songs",
	}
	var data map[string]object
	if err := rpc("video.VideoDataServer", "get_video_info_batch", map[string]any{"vidlist": vids, "required": required}, &data); err != nil {
		return nil, err
	}
	out := make(map[string]MVDetail, len(data))
	for vid, o := range data {
		d := MVDetail{
			Description: text(o, "desc"),
			Favorite:    number(o, "isfav") != 0,
			Uploader:    text(o, "uploader_nick"),
		}
		if mvs := parseMVs([]object{o}); len(mvs) == 1 {
			d.MV = mvs[0]
		}
		var related []int64
		if raw := o["related_songs"]; len(raw) > 0 && json.Unmarshal(raw, &related) == nil {
			d.RelatedSongs = related
		}
		out[vid] = d
	}
	return out, nil
}

// MVLink is one way QQ serves a video: its file, and when the address
// expires.
type MVLink struct {
	URL         string
	FreeflowURL string
	CommURL     string
	M3U8        string
	VKey        string
	Expire      int64
	Code        int64
	FileType    int64
	NewFileType int64
	Format      int64
	FileSize    int64
}

// MVStreams is the ways one video is served, in MP4 and HLS.
type MVStreams struct {
	MP4      []MVLink
	HLS      []MVLink
	Duration int64
}

// MVLinks returns where QQ serves the videos of the vids given, by vid.
func MVLinks(vids []string) (map[string]MVStreams, error) {
	guid := make([]byte, 16)
	if _, err := rand.Read(guid); err != nil {
		return nil, err
	}
	param := map[string]any{
		"vids": vids, "request_type": 10003, "guid": hex.EncodeToString(guid),
		"videoformat": 1, "format": 265, "dolby": 1, "use_new_domain": 1, "use_ipv6": 1,
	}
	var data map[string]struct {
		MP4      []object `json:"mp4"`
		HLS      []object `json:"hls"`
		Duration int64    `json:"duration"`
	}
	if err := rpc("music.stream.MvUrlProxy", "GetMvUrls", param, &data); err != nil {
		return nil, err
	}
	out := make(map[string]MVStreams, len(data))
	for vid, s := range data {
		streams := MVStreams{MP4: mvLinks(s.MP4), HLS: mvLinks(s.HLS), Duration: s.Duration}
		out[vid] = streams
	}
	return out, nil
}

func mvLinks(entries []object) []MVLink {
	var out []MVLink
	for _, o := range entries {
		link := MVLink{
			URL:         firstURL(o, "url"),
			FreeflowURL: firstURL(o, "freeflow_url"),
			CommURL:     firstURL(o, "comm_url"),
			M3U8:        text(o, "m3u8"),
			VKey:        text(o, "vkey"),
			Expire:      number(o, "expire"),
			Code:        number(o, "code"),
			FileType:    number(o, "filetype"),
			NewFileType: number(o, "newFileType"),
			Format:      number(o, "format"),
			FileSize:    number(o, "fileSize"),
		}
		if link.URL == "" {
			link.URL = link.FreeflowURL
		}
		if link.URL == "" {
			link.URL = link.CommURL
		}
		out = append(out, link)
	}
	return out
}

func firstURL(o object, key string) string {
	var urls []string
	if raw := o[key]; len(raw) > 0 && json.Unmarshal(raw, &urls) == nil && len(urls) > 0 {
		return urls[0]
	}
	return text(o, key)
}

// MVList returns a page of all videos QQ lists, and how many there are.
// area 15 is all and 8, 5, 6, 7, 4 are China, Hong Kong and Taiwan, Europe
// and America, Korea, Japan; version 7 is all, 8 videos, 13 lives, 14
// covers, 15 dances, 16 film, 17 shows, 18 children.
func MVList(area, version, order, page, num int) ([]MV, int, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"area": area, "version": version, "order": order, "start": (page - 1) * num, "size": num}
	var data object
	if err := rpc("MvService.MvInfoProServer", "GetAllocMvInfo", param, &data); err != nil {
		return nil, 0, err
	}
	return parseMVs(objects(data, "list")), int(number(data, "total")), nil
}
