package qqmusic

import "errors"

// SongRef is a song in a playlist write: its id and the type QQ reads it as.
type SongRef struct {
	ID   int64
	Type int
}

// ReorderPlaylistSongs reorders a playlist with the full list of its songs in
// their new order. QQ rejects a song the playlist does not hold with code
// 80095 rather than reporting a rejected write.
func ReorderPlaylistSongs(dirID, tid int64, songs []SongRef) (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	values, err := playlistSongRefs(songs)
	if err != nil {
		return false, err
	}
	param := map[string]any{"dirId": dirID, "tid": tid, "v_songPair": values}
	var data object
	if err := rpc("music.musicasset.PlaylistDetailWrite", "SeqSonglist", param, &data); err != nil {
		return false, err
	}
	return number(data, "retCode") == 0, nil
}

// AddPlaylistSongs appends songs to a playlist by its directory and playlist
// ids; directory 201 is the account's liked songs. It reports false when QQ
// says the songs are already there (code 80092).
func AddPlaylistSongs(dirID, tid int64, songs []SongRef) (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	values, err := playlistSongRefs(songs)
	if err != nil {
		return false, err
	}
	param := map[string]any{"dirId": dirID, "tid": tid, "bFmtUtf8": true, "v_songInfo": values}
	var data object
	if err := rpc("music.musicasset.PlaylistDetailWrite", "AddSonglist", param, &data); err != nil {
		if songAlreadyListed(err) {
			return false, nil
		}
		return false, err
	}
	return number(data, "retCode") == 0, nil
}

// DeletePlaylistSongs removes songs from a playlist by its directory and
// playlist ids; directory 201 is the account's liked songs. It reports false
// when QQ says the songs are not there (code 80092).
func DeletePlaylistSongs(dirID, tid int64, songs []SongRef) (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	values, err := playlistSongRefs(songs)
	if err != nil {
		return false, err
	}
	param := map[string]any{"dirId": dirID, "tid": tid, "bFmtUtf8": 1, "v_songInfo": values}
	var data object
	if err := rpc("music.musicasset.PlaylistDetailWrite", "DelSonglist", param, &data); err != nil {
		if songAlreadyListed(err) {
			return false, nil
		}
		return false, err
	}
	return number(data, "retCode") == 0, nil
}

func playlistSongRefs(songs []SongRef) ([]map[string]any, error) {
	values := make([]map[string]any, 0, len(songs))
	for _, song := range songs {
		if song.ID <= 0 {
			return nil, errors.New("qqmusic: invalid song ID")
		}
		values = append(values, map[string]any{"songId": song.ID, "songType": songlistWriteType(song.Type)})
	}
	return values, nil
}

// songlistWriteType maps a song's read type to the type a playlist write takes.
func songlistWriteType(t int) int {
	switch t {
	case 0, 1:
		return 0
	case 5, 111, 112, 113:
		return t
	default:
		return 1
	}
}

func songAlreadyListed(err error) bool {
	var protocol *ProtocolError
	return errors.As(err, &protocol) && protocol.Code == 80092
}
