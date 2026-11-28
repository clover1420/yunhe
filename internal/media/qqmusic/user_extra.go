package qqmusic

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// UserHomepage is the header of a listener's profile page.
type UserHomepage struct {
	EncryptedUin string
	Name         string
	Avatar       string
	Background   string
	UserType     int
	Singer       json.RawMessage
	Followed     bool
	TabDetail    json.RawMessage
}

// UserHomepageInfo reads a listener's profile header by encrypted UIN. QQ
// expects a signed-in identity here, so a placeholder one is sent when the
// listener has not logged in.
func UserHomepageInfo(euin string) (*UserHomepage, error) {
	s := sessionSnapshot()
	if !s.valid() {
		s = Session{MusicID: "1", MusicKey: "placeholder-musickey", LoginType: 1, EncryptUin: strings.Repeat("0", 32)}
	}
	var data object
	param := map[string]any{"uin": euin, "IsQueryTabDetail": 1}
	if err := rpcWithComm("music.UnifiedHomepage.UnifiedHomepageSrv", "GetHomepageHeader", param, &data, authenticatedComm(s), nil); err != nil {
		return nil, err
	}
	info := obj(data["Info"])
	base := obj(info["BaseInfo"])
	return &UserHomepage{
		EncryptedUin: text(base, "EncryptedUin"),
		Name:         text(base, "Name"),
		Avatar:       text(base, "Avatar"),
		Background:   text(base, "BackgroundImage"),
		UserType:     int(number(base, "UserType")),
		Singer:       info["Singer"],
		Followed:     boolField(info, "IsFollowed"),
		TabDetail:    data["TabDetail"],
	}, nil
}

// VipIdentity is the account's membership categories and their validity.
type VipIdentity struct {
	VIP           int
	HugeVip       int
	HugeVipStart  string
	HugeVipEnd    string
	YearFlag      int
	HugeYearFlag  int
	Twelve        int
	TwelveStart   string
	TwelveEnd     string
	ChildVip      int
	ExpVip        int
	GroupVipFlag  int
	GroupVipStart string
	GroupVipEnd   string
	CPLoverFlag   int
	CPLoverStart  string
	CPLoverEnd    string
	AdVipFlag     int
	Eight         int
	EightStart    string
	EightEnd      string
	Level         int
	NextLevel     int
	Icon          string
	PurchaseURL   string
}

// VipUserInfo is the account's membership summary: what it can do and when it
// expires.
type VipUserInfo struct {
	BuyURL     string
	MyVipURL   string
	Score      int64
	Expire     int64
	MusicLevel int
}

// VIPInfo is the signed-in account's membership, with QQ's own quotas.
type VIPInfo struct {
	AutoDownload  bool
	CanRenew      bool
	MaxPlaylists  int64
	MaxSongs      int64
	SongLimitHint string
	Super         bool
	Star          int
	StarStart     string
	StarEnd       string
	YearStar      int
	YearStarStart string
	YearStarEnd   string
	Identity      VipIdentity
	User          VipUserInfo
}

// UserVIPInfo reads the signed-in account's membership. iSuperVip alone is
// not enough to identify super VIP, so the HugeVip flag is used.
func UserVIPInfo() (*VIPInfo, error) {
	if _, err := signedSession(); err != nil {
		return nil, err
	}
	var data object
	if err := rpc("VipLogin.VipLoginInter", "vip_login_base", map[string]any{}, &data); err != nil {
		return nil, err
	}
	identity := obj(data["identity"])
	user := obj(data["userinfo"])
	return &VIPInfo{
		AutoDownload:  number(data, "auto_down", "autoDown", "autodown") != 0,
		CanRenew:      number(data, "canRenew") != 0,
		MaxPlaylists:  number(data, "max_dir_num", "maxDirNum", "maxdirnum"),
		MaxSongs:      number(data, "max_song_num", "maxSongNum", "maxsongnum"),
		SongLimitHint: text(data, "song_limit_msg", "songLimitMsg"),
		Super:         number(data, "svip") != 0,
		Star:          int(number(data, "star")),
		StarStart:     text(data, "starstart"),
		StarEnd:       text(data, "starend"),
		YearStar:      int(number(data, "ystar")),
		YearStarStart: text(data, "ystarstart"),
		YearStarEnd:   text(data, "ystarend"),
		Identity: VipIdentity{
			VIP:           int(number(identity, "vip")),
			HugeVip:       int(number(identity, "HugeVip")),
			HugeVipStart:  text(identity, "HugeVipStart"),
			HugeVipEnd:    text(identity, "HugeVipEnd"),
			YearFlag:      int(number(identity, "yearflag")),
			HugeYearFlag:  int(number(identity, "HugeYearFlag")),
			Twelve:        int(number(identity, "twelve")),
			TwelveStart:   text(identity, "twelveStart"),
			TwelveEnd:     text(identity, "twelveEnd"),
			ChildVip:      int(number(identity, "ChildVip")),
			ExpVip:        int(number(identity, "ExpVip")),
			GroupVipFlag:  int(number(identity, "GroupVipFlag")),
			GroupVipStart: text(identity, "GroupVipStart"),
			GroupVipEnd:   text(identity, "GroupVipEnd"),
			CPLoverFlag:   int(number(identity, "CPLoverFlag")),
			CPLoverStart:  text(identity, "CPLoverStart"),
			CPLoverEnd:    text(identity, "CPLoverEnd"),
			AdVipFlag:     int(number(identity, "AdVipFlag")),
			Eight:         int(number(identity, "eight")),
			EightStart:    text(identity, "eightStart"),
			EightEnd:      text(identity, "eightEnd"),
			Level:         int(number(identity, "level")),
			NextLevel:     int(number(identity, "nextlevel")),
			Icon:          text(identity, "icon"),
			PurchaseURL:   text(identity, "purchaseUrl"),
		},
		User: VipUserInfo{
			BuyURL:     text(user, "buy_url", "buyurl"),
			MyVipURL:   text(user, "my_vip_url", "myvipurl"),
			Score:      number(user, "score"),
			Expire:     number(user, "expire"),
			MusicLevel: int(number(user, "music_level", "musicLevel")),
		},
	}, nil
}

// RelationUser is a listener in one of QQ's social lists.
type RelationUser struct {
	MID         string
	EncryptUin  string
	Name        string
	Description string
	Avatar      string
	Fans        int64
	Followed    bool
}

// RelationList is one page of a listener's fans or follows.
type RelationList struct {
	Users       []RelationUser
	Total       int64
	HasMore     bool
	LastPos     string
	Message     string
	LockFlag    int
	LockMessage string
}

// UserFans returns a page of a listener's fans, by encrypted UIN.
func UserFans(euin string, page, num int) (*RelationList, error) {
	return userRelations("GetFansList", euin, page, num)
}

// UserFollowedSingers returns a page of the singers a listener follows.
func UserFollowedSingers(euin string, page, num int) (*RelationList, error) {
	return userRelations("GetFollowSingerList", euin, page, num)
}

// UserFollowedUsers returns a page of the listeners a listener follows.
func UserFollowedUsers(euin string, page, num int) (*RelationList, error) {
	return userRelations("GetFollowUserList", euin, page, num)
}

func userRelations(method, euin string, page, num int) (*RelationList, error) {
	if _, err := signedSession(); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"HostUin": euin, "From": (page - 1) * num, "Size": num}
	var data object
	if err := rpc("music.concern.RelationList", method, param, &data); err != nil {
		return nil, err
	}
	list := &RelationList{
		Total:       number(data, "Total"),
		HasMore:     rawBool(data["HasMore"]),
		LastPos:     text(data, "LastPos"),
		Message:     text(data, "Msg"),
		LockFlag:    int(number(data, "LockFlag")),
		LockMessage: text(data, "LockMsg"),
	}
	for _, o := range objects(data, "List") {
		list.Users = append(list.Users, RelationUser{
			MID:         text(o, "MID", "Mid"),
			EncryptUin:  text(o, "EncUin"),
			Name:        text(o, "Name"),
			Description: text(o, "Desc"),
			Avatar:      text(o, "AvatarUrl"),
			Fans:        number(o, "FanNum"),
			Followed:    boolField(o, "IsFollow"),
		})
	}
	return list, nil
}

// Friend is one of the signed-in account's friends.
type Friend struct {
	EncryptUin string
	Name       string
	Avatar     string
	Followed   bool
}

// FriendList is one page of the signed-in account's friends.
type FriendList struct {
	Friends []Friend
	HasMore bool
}

// UserFriends returns a page of the signed-in account's friends. page starts
// at 1.
func UserFriends(page, num int) (*FriendList, error) {
	if _, err := signedSession(); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	var data object
	param := map[string]any{"PageSize": num, "Page": page - 1}
	if err := rpc("music.homepage.Friendship", "GetFriendList", param, &data); err != nil {
		return nil, err
	}
	list := &FriendList{HasMore: rawBool(data["HasMore"])}
	for _, o := range objects(data, "Friends") {
		list.Friends = append(list.Friends, Friend{
			EncryptUin: text(o, "EncryptUin"),
			Name:       text(o, "UserName"),
			Avatar:     text(o, "AvatarUrl"),
			Followed:   boolField(o, "IsFollow"),
		})
	}
	return list, nil
}

// MusicGene is a listener's listening profile: their card, their report and
// the sort hints over it.
type MusicGene struct {
	UserInfoCard    json.RawMessage
	ListeningReport json.RawMessage
	SortArray       []int
	Self            bool
}

// UserMusicGene reads a listener's listening profile by encrypted UIN.
func UserMusicGene(euin string) (*MusicGene, error) {
	var data object
	if err := rpc("music.recommend.UserProfileSettingSvr", "GetProfileReport", map[string]any{"VisitAccount": euin}, &data); err != nil {
		return nil, err
	}
	gene := &MusicGene{
		UserInfoCard:    data["UserInfoCard"],
		ListeningReport: data["ListeningReport"],
		Self:            boolField(data, "IsVisitAccount"),
	}
	_ = json.Unmarshal(data["SortArray"], &gene.SortArray)
	return gene, nil
}

// FavoriteSongs is one page of a listener's liked songs.
type FavoriteSongs struct {
	Songs []Track
	Total int64
	More  bool
}

// UserFavoriteSongs reads a page of a listener's liked songs by encrypted UIN.
func UserFavoriteSongs(euin string, page, num int) (*FavoriteSongs, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{
		"disstid": 0, "dirid": 201, "tag": 1,
		"song_begin": (page - 1) * num, "song_num": num,
		"userinfo": 1, "orderlist": 1, "enc_host_uin": euin,
	}
	var data object
	if err := rpc("music.srfDissInfo.DissInfo", "CgiGetDiss", param, &data); err != nil {
		return nil, err
	}
	if code := number(data, "code"); code != 0 {
		return nil, fmt.Errorf("qqmusic: liked songs code %d", code)
	}
	return &FavoriteSongs{
		Songs: convertTracks(objects(data, "songlist")),
		Total: number(data, "total_song_num"),
		More:  rawBool(data["hasmore"]),
	}, nil
}

// FavoritePlaylistItem is a playlist a listener saved from another creator.
type FavoritePlaylistItem struct {
	ID          int64
	DirID       int64
	Name        string
	Cover       string
	Description string
	Songs       int64
	Plays       int64
	Uin         string
	Nickname    string
	CreatedAt   int64
	UpdatedAt   int64
	SavedAt     int64
}

// FavoritePlaylists is one page of a listener's saved playlists.
type FavoritePlaylists struct {
	Playlists []FavoritePlaylistItem
	Total     int64
	More      bool
	Hidden    bool
}

// UserFavoritePlaylists reads a page of the playlists a listener saved. QQ
// names this request's account field uin, not euin.
func UserFavoritePlaylists(euin string, page, num int) (*FavoritePlaylists, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"uin": euin, "offset": (page - 1) * num, "size": num}
	var data object
	if err := rpc("music.musicasset.PlaylistFavRead", "CgiGetPlaylistFavInfo", param, &data); err != nil {
		return nil, err
	}
	pageData := &FavoritePlaylists{
		Total:  number(data, "total"),
		More:   rawBool(data["hasmore"]),
		Hidden: boolField(data, "hide"),
	}
	for _, o := range objects(data, "v_list") {
		pageData.Playlists = append(pageData.Playlists, favoritePlaylistItem(o))
	}
	return pageData, nil
}

func favoritePlaylistItem(o object) FavoritePlaylistItem {
	return FavoritePlaylistItem{
		ID:          number(o, "id", "tid", "dissid"),
		DirID:       number(o, "dirid", "dirId"),
		Name:        text(o, "title", "dissname", "name", "dirName"),
		Cover:       text(o, "picurl", "cover", "logo", "picUrl"),
		Description: text(o, "desc", "description"),
		Songs:       number(o, "songnum", "songNum", "song_cnt"),
		Plays:       number(o, "listennum", "playCnt", "play_cnt"),
		Uin:         text(o, "uin"),
		Nickname:    text(o, "nickname", "nick"),
		CreatedAt:   number(o, "createtime", "createTime"),
		UpdatedAt:   number(o, "updateTime", "updatetime"),
		SavedAt:     number(o, "orderTime"),
	}
}

// FavoriteAlbumItem is an album a listener saved.
type FavoriteAlbumItem struct {
	Album
	Artists     []Artist
	Songs       int64
	PublishTime int64
	SavedAt     int64
	Status      int
	Location    int
}

// FavoriteAlbums is one page of a listener's saved albums.
type FavoriteAlbums struct {
	Albums []FavoriteAlbumItem
	Total  int64
	More   bool
	Hidden bool
}

// UserFavoriteAlbums reads a page of the albums a listener saved. Unlike
// playlists, QQ names this request's account field euin.
func UserFavoriteAlbums(euin string, page, num int) (*FavoriteAlbums, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"euin": euin, "offset": (page - 1) * num, "size": num}
	var data object
	if err := rpc("music.musicasset.AlbumFavRead", "CgiGetAlbumFavInfo", param, &data); err != nil {
		return nil, err
	}
	pageData := &FavoriteAlbums{
		Total:  number(data, "total"),
		More:   rawBool(data["hasmore"]),
		Hidden: boolField(data, "hide"),
	}
	for _, o := range objects(data, "v_list") {
		item := FavoriteAlbumItem{
			Album:       convertAlbum(o),
			Songs:       number(o, "songnum"),
			PublishTime: number(o, "pubtime"),
			SavedAt:     number(o, "ordertime"),
			Status:      int(number(o, "status")),
			Location:    int(number(o, "loc")),
		}
		for _, s := range objects(o, "v_singer", "singer", "singers") {
			item.Artists = append(item.Artists, convertArtist(s))
		}
		pageData.Albums = append(pageData.Albums, item)
	}
	return pageData, nil
}

// FavoriteMVItem is a music video a listener saved.
type FavoriteMVItem struct {
	MV
	Singers   []Artist
	SingerID  int64
	SingerMID string
	Singer    string
	Status    int
}

// FavoriteMVs is the signed-in account's saved music videos. QQ pages this
// request by page index, not by an item count.
type FavoriteMVs struct {
	MVs  []FavoriteMVItem
	Code int
	Msg  string
}

// UserFavoriteMVs reads a page of the account's saved music videos by
// encrypted UIN. page starts at 1.
func UserFavoriteMVs(euin string, page, num int) (*FavoriteMVs, error) {
	if _, err := signedSession(); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"encuin": euin, "pagesize": num, "num": page - 1}
	var data object
	if err := rpc("music.musicasset.MVFavRead", "getMyFavMV_v2", param, &data); err != nil {
		return nil, err
	}
	pageData := &FavoriteMVs{Code: int(number(data, "code")), Msg: text(data, "msg")}
	for _, o := range objects(data, "mvlist") {
		item := FavoriteMVItem{
			MV: MV{
				ID:          number(o, "id", "sid", "mvid", "singerId"),
				VID:         text(o, "vid"),
				Name:        text(o, "name", "mvname", "title"),
				Title:       text(o, "title", "title_main", "name"),
				Cover:       text(o, "picUrl"),
				Plays:       number(o, "playcount"),
				PublishDate: number(o, "publish_date"),
			},
			SingerID:  number(o, "singerId"),
			SingerMID: text(o, "singerMid"),
			Singer:    text(o, "singerName"),
			Status:    int(number(o, "status")),
		}
		for _, s := range objects(o, "v_singer", "singers") {
			item.Singers = append(item.Singers, convertArtist(s))
		}
		pageData.MVs = append(pageData.MVs, item)
	}
	return pageData, nil
}

// CreatedPlaylistItem is a playlist its owner made.
type CreatedPlaylistItem struct {
	ID           int64
	DirID        int64
	Name         string
	Cover        string
	Description  string
	Songs        int64
	Plays        int64
	CreatedAt    int64
	UpdatedAt    int64
	Uin          string
	Nickname     string
	Favorites    int64
	CommentCount int64
}

// CreatedPlaylists is the playlists a listener made, and whether QQ returned
// them all.
type CreatedPlaylists struct {
	Playlists []CreatedPlaylistItem
	Total     int64
	Finished  bool
}

// UserCreatedPlaylists reads the playlists a listener made, by numeric UIN.
func UserCreatedPlaylists(uin int64) (*CreatedPlaylists, error) {
	var data object
	if err := rpc("music.musicasset.PlaylistBaseRead", "GetPlaylistByUin", map[string]any{"uin": strconv.FormatInt(uin, 10)}, &data); err != nil {
		return nil, err
	}
	playlists := &CreatedPlaylists{
		Total:    number(data, "total"),
		Finished: boolField(data, "bFinish"),
	}
	for _, o := range objects(data, "v_playlist") {
		playlists.Playlists = append(playlists.Playlists, CreatedPlaylistItem{
			ID:           number(o, "id", "tid", "dissid"),
			DirID:        number(o, "dirid", "dirId"),
			Name:         text(o, "title", "dissname", "name", "dirName"),
			Cover:        text(o, "picurl", "cover", "logo", "picUrl"),
			Description:  text(o, "desc", "description"),
			Songs:        number(o, "songnum", "songNum", "song_cnt"),
			Plays:        number(o, "listennum", "playCnt", "play_cnt"),
			CreatedAt:    number(o, "createTime"),
			UpdatedAt:    number(o, "updateTime"),
			Uin:          text(o, "uin"),
			Nickname:     text(o, "nick"),
			Favorites:    number(o, "fav_cnt"),
			CommentCount: number(o, "comment_cnt"),
		})
	}
	return playlists, nil
}

// DislikeItem is a song, singer or style the account asked not to hear.
type DislikeItem struct {
	ID    string
	Name  string
	Image string
	Type  int
	Time  int64
}

// DislikePage is one page of the account's dislikes for one kind of entity.
type DislikePage struct {
	Singers []DislikeItem
	Songs   []DislikeItem
	Styles  []DislikeItem
	Page    int
	Token   string
	RetCode int
	Message string
}

// DislikeList reads a page of the account's dislikes. cmd 2 is singers, 3
// songs and 4 styles; page starts at 1; lastID continues from the previous
// page's last item.
func DislikeList(cmd, page, lastID int) (*DislikePage, error) {
	s, err := signedSession()
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	param := map[string]any{"Cmd": cmd, "Page": page}
	switch cmd {
	case 2:
		if lastID != 0 {
			param["SingersLastid"] = lastID
		}
	case 3:
		if lastID != 0 {
			param["SongLastid"] = lastID
		}
	case 4:
		if lastID != 0 {
			param["StyleLastid"] = lastID
		}
	default:
		return nil, errors.New("不喜欢列表的类型应为 2（歌手）、3（歌曲）或 4（风格）")
	}
	var data object
	if err := rpcSigned("music.feedback.FeedbackBlack", "GetDislikeList", param, authenticatedComm(s), &data); err != nil {
		return nil, err
	}
	result := &DislikePage{
		Page:    int(number(data, "Page")),
		Token:   text(data, "Token"),
		RetCode: int(number(data, "Retcode")),
		Message: text(data, "Msg"),
	}
	for _, key := range []struct {
		name  string
		items *[]DislikeItem
	}{{"Singers", &result.Singers}, {"Songs", &result.Songs}, {"Styles", &result.Styles}} {
		for _, o := range objects(data, key.name) {
			*key.items = append(*key.items, DislikeItem{
				ID:    text(o, "ID"),
				Name:  text(o, "Name"),
				Image: text(o, "Img"),
				Type:  int(number(o, "IdType")),
				Time:  number(o, "Time"),
			})
		}
	}
	return result, nil
}

// AddDislikes asks QQ to stop playing songs, singers or styles. idType is 1
// for songs, 2 for singers and 3 for styles.
func AddDislikes(idType int, ids []int64) (bool, error) {
	return writeDislikes("AddDislike", idType, ids)
}

// RemoveDislikes lifts a dislike added with AddDislikes.
func RemoveDislikes(idType int, ids []int64) (bool, error) {
	return writeDislikes("CancelDislike", idType, ids)
}

func writeDislikes(method string, idType int, ids []int64) (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	key := map[int]string{1: "Songs", 2: "Singers", 3: "Styles"}[idType]
	if key == "" {
		return false, errors.New("不喜欢的类型应为 1（歌曲）、2（歌手）或 3（风格）")
	}
	values := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		values = append(values, map[string]any{"ID": strconv.FormatInt(id, 10), "IdType": idType})
	}
	var data struct {
		Retcode *int `json:"Retcode"`
	}
	if err := rpc("music.feedback.FeedbackBlack", method, map[string]any{key: values}, &data); err != nil {
		return false, err
	}
	return data.Retcode != nil && *data.Retcode == 0, nil
}

// ClearDislikedSongs removes every disliked song. QQ first hands out a token
// and then clears the list with it.
func ClearDislikedSongs() (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	var first struct {
		Token string `json:"Token"`
	}
	if err := rpc("music.feedback.FeedbackBlack", "CancelAllDislike", map[string]any{"ISOnlyGetToken": true}, &first); err != nil {
		return false, err
	}
	var second struct {
		Retcode *int `json:"Retcode"`
	}
	if err := rpc("music.feedback.FeedbackBlack", "CancelAllDislike", map[string]any{"DelType": 3, "Token": first.Token}, &second); err != nil {
		return false, err
	}
	return second.Retcode != nil && *second.Retcode == 0, nil
}

// FavoritePlaylist saves another creator's playlist to the account's library.
func FavoritePlaylist(id int64) (bool, error) {
	s, err := signedSession()
	if err != nil {
		return false, err
	}
	if s.EncryptUin == "" {
		return false, errors.New("QQ 音乐登录凭证缺少账号标识，请重新登录")
	}
	if id <= 0 {
		return false, errors.New("qqmusic: invalid playlist ID")
	}
	return playlistFavoriteWrite("FavPlaylist", s.EncryptUin, id)
}

// RemoveFavoritePlaylist removes another creator's playlist from the account's
// library.
func RemoveFavoritePlaylist(id int64) (bool, error) {
	s, err := signedSession()
	if err != nil {
		return false, err
	}
	if s.EncryptUin == "" {
		return false, errors.New("QQ 音乐登录凭证缺少账号标识，请重新登录")
	}
	if id <= 0 {
		return false, errors.New("qqmusic: invalid playlist ID")
	}
	return playlistFavoriteWrite("CancelFavPlaylist", s.EncryptUin, id)
}

func playlistFavoriteWrite(method, uin string, id int64) (bool, error) {
	var data struct {
		Result *int              `json:"result"`
		Failed []json.RawMessage `json:"v_failedPlaylistId"`
	}
	if err := rpc("music.musicasset.PlaylistFavWrite", method, map[string]any{"uin": uin, "v_playlistId": []int64{id}}, &data); err != nil {
		return false, err
	}
	if data.Result == nil || *data.Result != 0 {
		return false, nil
	}
	want := strconv.FormatInt(id, 10)
	for _, raw := range data.Failed {
		var failed any
		if json.Unmarshal(raw, &failed) == nil && fmt.Sprint(failed) == want {
			return false, nil
		}
	}
	return true, nil
}
