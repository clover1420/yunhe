package qqmusic

// TopPreviewSong is one of the songs a chart's card shows before its page.
type TopPreviewSong struct {
	Rank       int
	SongID     int64
	Name       string
	SingerName string
	SingerMID  string
	AlbumMID   string
	Cover      string
	MVID       int64
}

// TopList is a chart: what it is, and how it stands.
type TopList struct {
	ID          int64
	Title       string
	TitleDetail string
	TitleSub    string
	Intro       string
	Period      string
	UpdateTime  string
	ListenNum   int64
	TotalNum    int64
	FrontPic    string
	HeadPic     string
	Songs       []TopPreviewSong
}

// TopCategory is a group of charts, as QQ's chart page lists them.
type TopCategory struct {
	ID    int64
	Name  string
	Lists []TopList
}

// TopCategories returns QQ's charts, in their groups.
func TopCategories() ([]TopCategory, error) {
	var data object
	if err := rpc("music.musicToplist.Toplist", "GetAll", map[string]any{}, &data); err != nil {
		return nil, err
	}
	var out []TopCategory
	for _, g := range objects(data, "group") {
		c := TopCategory{ID: number(g, "groupId"), Name: text(g, "groupName")}
		for _, t := range objects(g, "toplist") {
			c.Lists = append(c.Lists, parseTopList(t))
		}
		out = append(out, c)
	}
	return out, nil
}

func parseTopList(o object) TopList {
	t := TopList{
		ID:          number(o, "topId"),
		Title:       text(o, "title"),
		TitleDetail: text(o, "titleDetail"),
		TitleSub:    text(o, "titleSub"),
		Intro:       text(o, "intro"),
		Period:      text(o, "period"),
		UpdateTime:  text(o, "updateTime"),
		ListenNum:   number(o, "listenNum"),
		TotalNum:    number(o, "totalNum"),
		FrontPic:    text(o, "frontPicUrl"),
		HeadPic:     text(o, "headPicUrl"),
	}
	for _, s := range objects(o, "song") {
		t.Songs = append(t.Songs, TopPreviewSong{
			Rank:       int(number(s, "rank")),
			SongID:     number(s, "songId"),
			Name:       text(s, "title", "name"),
			SingerName: text(s, "singerName"),
			SingerMID:  text(s, "singerMid"),
			AlbumMID:   text(s, "albumMid"),
			Cover:      text(s, "cover"),
			MVID:       number(s, "mvid"),
		})
	}
	return t
}

// TopDetail is a chart's page: what it is, and its songs.
type TopDetail struct {
	Info  TopList
	Songs []Track
}

// TopListDetail returns a chart's page, page from 1. withTags asks for the
// tags of each song, which QQ takes as a boolean.
func TopListDetail(topID int64, page, num int, withTags bool) (*TopDetail, error) {
	if page < 1 {
		page = 1
	}
	if num <= 0 {
		num = 10
	}
	param := map[string]any{"topId": topID, "offset": (page - 1) * num, "num": num}
	if withTags {
		param["withTags"] = true
	}
	var data object
	if err := rpc("music.musicToplist.Toplist", "GetDetail", param, &data); err != nil {
		return nil, err
	}
	d := &TopDetail{Info: parseTopList(obj(data["data"]))}
	d.Songs = convertTracks(objects(data, "songInfoList"))
	return d, nil
}
