package qqmusic

import "strconv"

// AIDictItem is a word in a lyric that QQ's AI explains for the listener.
type AIDictItem struct {
	Phrase      string `json:"phrase"`
	Explain     string `json:"explain"`
	Lyric       string `json:"lyricText"`
	Translation string `json:"translation"`
	Timestamp   string `json:"timestamp"`
}

// AIDict returns the words of a song's lyric that QQ's AI explains.
func AIDict(songID int64) ([]AIDictItem, error) {
	var data object
	if err := rpc("music.musichallSong.PlayLyricInfo", "GetAIDictInfo", map[string]any{"songID": songID}, &data); err != nil {
		return nil, err
	}
	entries := objects(data, "dictList", "dict_list")
	out := make([]AIDictItem, 0, len(entries))
	for _, o := range entries {
		stamp := text(o, "lyric_timestamp", "lyricTimestamp")
		if stamp == "" {
			if n := number(o, "lyric_timestamp", "lyricTimestamp"); n != 0 {
				stamp = strconv.FormatInt(n, 10)
			}
		}
		out = append(out, AIDictItem{
			Phrase:      text(o, "phrase"),
			Explain:     text(o, "explain"),
			Lyric:       text(o, "lyric_text", "lyricText"),
			Translation: text(o, "trans_lyric_text", "transLyricText"),
			Timestamp:   stamp,
		})
	}
	return out, nil
}

// HasAIDict says whether a song's lyric has words QQ's AI explains.
func HasAIDict(songID int64) (bool, error) {
	var data struct {
		Exists bool `json:"exists"`
	}
	err := rpc("music.musichallSong.PlayLyricInfo", "IsAIDictExists", map[string]any{"songID": songID}, &data)
	return data.Exists, err
}

// MultiStyleLyric is a song's lyric translated in a style: poetically, or
// into a dialect.
type MultiStyleLyric struct {
	Style     int    `json:"style"`
	Name      string `json:"styleName"`
	Lyric     string `json:"lyric"`
	Timestamp int64  `json:"timestamp"`
}

// MultiStyleLyrics returns a song's lyric in the styles QQ translates it
// into, decrypted where it comes encrypted.
func MultiStyleLyrics(songID int64) ([]MultiStyleLyric, error) {
	var data struct {
		Lyrics []MultiStyleLyric `json:"lyrics"`
	}
	if err := rpc("music.musichallSong.PlayLyricInfo", "BatchGetMultiStyleTransLyric", map[string]any{"songID": songID}, &data); err != nil {
		return nil, err
	}
	for i := range data.Lyrics {
		if plain, err := decodeQRC(data.Lyrics[i].Lyric); err == nil {
			data.Lyrics[i].Lyric = plain
		}
	}
	return data.Lyrics, nil
}

// HasSingingAnnotations says whether a song's lyric carries the notes the
// singing view shows: where to breathe, and how high each phrase goes.
func HasSingingAnnotations(songID int64) (bool, error) {
	var data struct {
		Has bool `json:"hasSingingAnnotationsLyric"`
	}
	param := map[string]any{"songID": songID, "needNum": false}
	err := rpc("music.musichallSong.PlayLyricInfo", "GetSingingAnnotationsInfo", param, &data)
	return data.Has, err
}
