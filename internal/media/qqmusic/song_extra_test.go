package qqmusic

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The zzc signature matches the reference implementation, byte for byte.
func TestZZCSign(t *testing.T) {
	body := []byte(`{"comm":{"format":"json","g_tk":5381,"inCharset":"utf-8","needNewCode":1,"notice":0,"outCharset":"utf-8","platform":"h5","uin":""},"req_0":{"method":"GetChongChongSheetMusic","module":"music.mir.SheetMusicSvr","param":{"begin":0,"end":100,"scoreType":-1,"songMid":"003w2xz20QlUZt","ttype":1}}}`)
	const want = "zzc966e070zll08asrlzcwdbqs5nddqyfkc8072a0777"
	if got := zzcSign(body); got != want {
		t.Fatalf("zzcSign() = %q, want %q", got, want)
	}
}

// The extras decode the fields QQ returns, and a sheet service's 10007
// means no score rather than a failure.
func TestSongExtraDecoding(t *testing.T) {
	withQQServer(t, func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Req struct {
				Module, Method string
				Param          map[string]any
			} `json:"req_0"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		switch request.Req.Method {
		case "GetSongFansNumberById":
			if _, ok := request.Req.Param["v_songId"]; !ok {
				t.Error("favorites request lost v_songId")
			}
			rpcReply(w, map[string]any{"m_numbers": map[string]int64{"100": 12345}})
		case "GetSongLabels":
			rpcReply(w, map[string]any{"labels": []map[string]any{{"id": 7, "tagTxt": "国风", "tagType": 2}}})
		case "GetSimilarSongs":
			rpcReply(w, map[string]any{"vecSongNew": []map[string]any{{
				"title_content": "相似歌曲",
				"songs": []map[string]any{{"track": map[string]any{
					"id": 9, "mid": "m9", "name": "N", "interval": 180,
					"singer": []map[string]any{{"id": 1, "mid": "s1", "name": "S"}},
					"album":  map[string]any{"id": 2, "mid": "al", "name": "A"},
				}}},
			}}})
		case "GetRelatedPlaylist":
			rpcReply(w, map[string]any{
				"hasMore": 1,
				"vecPlaylistNew": []map[string]any{{"playlists": []map[string]any{{
					"tid": 55, "title": "歌单", "picurl": "p", "songnum": 12, "creator": "C",
				}}}},
			})
		case "GetSongRelatedMv":
			rpcReply(w, map[string]any{
				"hasmore": 1,
				"list": []map[string]any{{
					"mvid": 77, "mvname": "MV", "picurl": "p", "playcnt": 9,
					"singers": []map[string]any{{"id": 1, "mid": "s1", "name": "S"}},
				}},
			})
		case "GetChongChongSheetMusic":
			rpcReply(w, map[string]any{"result": []map[string]any{{"scoreMID": "sm", "scoreName": "Score", "picURLs": []string{"p1"}}}})
		case "GetMoreSheetMusic":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "req_0": map[string]any{"code": 10007}})
		default:
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
	})

	favorites, err := SongFavorites([]int64{100})
	if err != nil || favorites[100] != 12345 {
		t.Fatalf("SongFavorites = %v, %v", favorites, err)
	}
	labels, err := SongLabels(100)
	if err != nil || len(labels) != 1 || labels[0].Text != "国风" {
		t.Fatalf("SongLabels = %v, %v", labels, err)
	}
	groups, err := SimilarSongs(100)
	if err != nil || len(groups) != 1 || len(groups[0].Songs) != 1 || groups[0].Songs[0].MID != "m9" {
		t.Fatalf("SimilarSongs = %v, %v", groups, err)
	}
	lists, more, next, err := RelatedPlaylists(100, nil)
	if err != nil || !more || len(lists) != 1 || lists[0].Name != "歌单" || len(next) != 1 {
		t.Fatalf("RelatedPlaylists = %v, %v, %v, %v", lists, more, next, err)
	}
	mvs, more, err := RelatedMVs(100, 0)
	if err != nil || !more || len(mvs) != 1 || mvs[0].ID != 77 || len(mvs[0].Singers) != 1 {
		t.Fatalf("RelatedMVs = %v, %v, %v", mvs, more, err)
	}
	sheets, err := Sheets("mid", SheetChongChong)
	if err != nil || len(sheets) != 1 || sheets[0].MID != "sm" {
		t.Fatalf("Sheets(chongchong) = %v, %v", sheets, err)
	}
	empty, err := Sheets("mid", SheetUploaded)
	if err != nil || len(empty) != 0 {
		t.Fatalf("Sheets(empty) = %v, %v", empty, err)
	}
}
