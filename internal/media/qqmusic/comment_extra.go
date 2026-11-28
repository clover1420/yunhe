package qqmusic

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strconv"
)

// CommentBiz identifies what a comment thread belongs to.
const (
	CommentBizSong         = 1
	CommentBizAlbum        = 2
	CommentBizPlaylist     = 3
	CommentBizMV           = 4
	CommentBizSpecialAudio = 15
)

// CommentIcon is a badge QQ shows over a comment thread's counter.
type CommentIcon struct {
	Text     string
	UniqueID string
	Type     int
	Comment  string
	Dynamic  bool
}

// CommentCountInfo is what QQ answers when asked how many comments a thread
// holds, including the wording of the count its pages show.
type CommentCountInfo struct {
	BizType      int
	BizID        string
	BizSubType   int
	Count        int64
	CountVersion string
	CountView    string
	RelatedID    string
	Tip          string
	Icons        []CommentIcon
	TabType      int
}

// CommentCount returns how many comments a thread holds. bizSubType may be 0
// to let QQ supply its own, as it does for songs.
func CommentCount(bizID int64, bizType, bizSubType int) (*CommentCountInfo, error) {
	if bizType <= 0 {
		bizType = CommentBizSong
	}
	request := map[string]any{"biz_id": strconv.FormatInt(bizID, 10), "biz_type": bizType}
	sub := bizSubType
	if sub == 0 && bizType == CommentBizSong {
		sub = 2
	}
	if sub > 0 {
		request["biz_sub_type"] = sub
	}
	var data struct {
		Response *struct {
			BizType    int    `json:"biz_type"`
			BizID      string `json:"biz_id"`
			BizSubType int    `json:"biz_sub_type"`
			Count      int64  `json:"count"`
			CountVer   string `json:"count_ver"`
			CountView  string `json:"count_view"`
			RelatedID  string `json:"related_id"`
			Tip        string `json:"tip"`
			Icons      []struct {
				Text     string `json:"txt"`
				UniqueID string `json:"unique_id"`
				Type     int    `json:"type"`
				Comment  string `json:"cmid"`
				Dynamic  bool   `json:"is_dynamic"`
			} `json:"icon_list"`
		} `json:"response"`
		CmTabType int `json:"cmTabType"`
	}
	if err := rpc("music.globalComment.CommentCountSrv", "GetCmCount", map[string]any{"request": request}, &data); err != nil {
		return nil, err
	}
	if data.Response == nil {
		return nil, errors.New("QQ 音乐未返回评论数量")
	}
	info := &CommentCountInfo{
		BizType:      data.Response.BizType,
		BizID:        data.Response.BizID,
		BizSubType:   data.Response.BizSubType,
		Count:        data.Response.Count,
		CountVersion: data.Response.CountVer,
		CountView:    data.Response.CountView,
		RelatedID:    data.Response.RelatedID,
		Tip:          data.Response.Tip,
		TabType:      data.CmTabType,
	}
	for _, icon := range data.Response.Icons {
		info.Icons = append(info.Icons, CommentIcon{
			Text: icon.Text, UniqueID: icon.UniqueID, Type: icon.Type,
			Comment: icon.Comment, Dynamic: icon.Dynamic,
		})
	}
	return info, nil
}

// CommentItem is one comment as QQ's thread pages show it.
type CommentItem struct {
	ID         string
	SeqNo      string
	Nickname   string
	Avatar     string
	EncryptUin string
	Content    string
	Time       int64
	Likes      int64
	Replies    int64
	Praised    bool
	Self       bool
	State      int
	HotScore   string
	RecScore   string
	SongID     int64
	SongName   string
	Singers    string
	Timeline   []json.RawMessage
	HashTags   []json.RawMessage
	Tails      []json.RawMessage
	Icons      []json.RawMessage
	VIP        json.RawMessage
	SubReplies []json.RawMessage
}

// CommentPage is one page of a thread's comments. NextSeqNo is the cursor the
// following page asks for.
type CommentPage struct {
	Comments      []CommentItem
	Total         int64
	HasMore       bool
	NextSeqNo     string
	TotalComments int64
	Tip           string
	H5Page        string
	HasTimeline   bool
	Shares        int64
	SubCode       int
}

type commentListData struct {
	CommentList *struct {
		Comments   []object
		HasMore    json.RawMessage
		NextOffset int64
		Total      int64
	} `json:"CommentList"`
	TotalCmNum    int64  `json:"TotalCmNum"`
	CommentTip    string `json:"CommentTip"`
	CommentH5Page string `json:"CommentH5Page"`
	HasTsCm       int    `json:"HasTsCm"`
	ShareCnt      int64  `json:"ShareCnt"`
	Msg           string `json:"Msg"`
	SubCode       int    `json:"SubCode"`
}

// HotComments returns a page of a thread's most-liked comments. page starts at
// 1; pass the previous page's NextSeqNo as lastSeqNo to continue. bizSubType 0
// leaves the field out, as these endpoints do.
func HotComments(bizID int64, page, size int, lastSeqNo string, bizType, bizSubType int) (*CommentPage, error) {
	return readCommentList("GetHotCommentList", bizID, page, size, lastSeqNo, bizType, bizSubType, func(param map[string]any) {
		param["HotType"], param["WithAirborne"] = 1, 0
	})
}

// NewComments returns a thread's latest comments, paged like HotComments.
func NewComments(bizID int64, page, size int, lastSeqNo string, bizType, bizSubType int) (*CommentPage, error) {
	return readCommentList("GetNewCommentList", bizID, page, size, lastSeqNo, bizType, bizSubType, func(param map[string]any) {
		param["HashTagID"], param["SelfSeeEnable"], param["AudioEnable"] = "", 1, 1
	})
}

// RecommendComments returns the comments QQ recommends for a thread, paged
// like HotComments.
func RecommendComments(bizID int64, page, size int, lastSeqNo string, bizType, bizSubType int) (*CommentPage, error) {
	return readCommentList("GetRecCommentList", bizID, page, size, lastSeqNo, bizType, bizSubType, func(param map[string]any) {
		param["Flag"], param["CmListUIVer"], param["AudioEnable"] = 1, 1, 1
	})
}

func readCommentList(method string, bizID int64, page, size int, lastSeqNo string, bizType, bizSubType int, extra func(map[string]any)) (*CommentPage, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if bizType <= 0 {
		bizType = CommentBizSong
	}
	param := map[string]any{
		"BizType": bizType, "BizId": strconv.FormatInt(bizID, 10),
		"PageSize": size, "PageNum": page - 1, "LastCommentSeqNo": lastSeqNo,
		"PicEnable": 1,
	}
	if bizSubType > 0 {
		param["BizSubType"] = bizSubType
	}
	extra(param)
	var data commentListData
	if err := rpc("music.globalComment.CommentRead", method, param, &data); err != nil {
		return nil, err
	}
	if data.CommentList == nil {
		return nil, errors.New("QQ 音乐未返回评论列表")
	}
	result := &CommentPage{
		Comments:      decodeCommentItems(data.CommentList.Comments),
		Total:         data.CommentList.Total,
		HasMore:       rawBool(data.CommentList.HasMore),
		TotalComments: data.TotalCmNum,
		Tip:           data.CommentTip,
		H5Page:        data.CommentH5Page,
		HasTimeline:   data.HasTsCm == 1,
		Shares:        data.ShareCnt,
		SubCode:       data.SubCode,
	}
	if n := len(data.CommentList.Comments); n > 0 {
		result.NextSeqNo = text(data.CommentList.Comments[n-1], "SeqNo")
	}
	return result, nil
}

// MomentComment is one comment on a song's timeline.
type MomentComment struct {
	ID          string
	SeqNo       string
	Content     string
	EncryptUin  string
	Time        int64
	Likes       int64
	Replies     int64
	State       int
	Self        bool
	Location    string
	Phone       string
	Picture     string
	PictureSize string
	Timeline    []json.RawMessage
	HashTags    []json.RawMessage
	Tails       []json.RawMessage
}

// MomentCommentPage is one page of timeline comments. NextPos is the cursor
// the following page asks for.
type MomentCommentPage struct {
	Comments       []MomentComment
	HasMore        bool
	NextPos        string
	Hint           string
	PrevListLoaded int
	Extras         map[string]json.RawMessage
}

// MomentComments returns a page of a song's timeline comments. Pass the
// previous page's NextPos as lastPos to continue.
func MomentComments(bizID int64, size int, lastPos string, bizType, bizSubType int) (*MomentCommentPage, error) {
	if size <= 0 {
		size = 15
	}
	if bizType <= 0 {
		bizType = CommentBizSong
	}
	param := map[string]any{
		"LastPos": lastPos, "HashTagID": "", "SeekTs": -1, "Size": size,
		"BizType": bizType, "BizId": strconv.FormatInt(bizID, 10),
	}
	if bizSubType > 0 {
		param["BizSubType"] = bizSubType
	}
	var data struct {
		Comments   []object                   `json:"CmList"`
		HasMore    json.RawMessage            `json:"HasMore"`
		NextPos    string                     `json:"NextPos"`
		Hint       string                     `json:"Hint"`
		PrevLoaded int                        `json:"PrevListLoaded"`
		Extras     map[string]json.RawMessage `json:"MapCmExt"`
	}
	if err := rpc("music.globalComment.SongTsComment", "GetSongTsCmList", param, &data); err != nil {
		return nil, err
	}
	page := &MomentCommentPage{
		HasMore:        rawBool(data.HasMore),
		NextPos:        data.NextPos,
		Hint:           data.Hint,
		PrevListLoaded: data.PrevLoaded,
		Extras:         data.Extras,
	}
	for _, o := range data.Comments {
		id := text(o, "CmId")
		if id == "" {
			continue
		}
		page.Comments = append(page.Comments, MomentComment{
			ID: id, SeqNo: text(o, "SeqNo"), Content: html.UnescapeString(text(o, "Content")),
			EncryptUin: text(o, "EncryptUin"), Time: number(o, "PubTime"),
			Likes: number(o, "PraiseNum"), Replies: number(o, "ReplyCnt"),
			State: int(number(o, "State")), Self: boolField(o, "IsSelf"),
			Location: text(o, "Location"), Phone: text(o, "PhoneType"),
			Picture: text(o, "Pic"), PictureSize: text(o, "PicSize"),
			Timeline: rawList(o, "SongTsElems"), HashTags: rawList(o, "HashTagList"),
			Tails: rawList(o, "LittleTails"),
		})
	}
	return page, nil
}

// AddedComment is the comment QQ accepted or the captcha it demands instead.
type AddedComment struct {
	SubCode   int
	Message   string
	ID        string
	ParentID  string
	Floor     int
	VerifyURL string
}

// AddComment posts a comment on a thread. replyTo is the comment being replied
// to, or empty for a top-level comment. A non-zero SubCode means QQ refused
// the comment; VerifyURL may then hold a captcha.
func AddComment(bizID int64, content, replyTo string, bizType, bizSubType int) (*AddedComment, error) {
	if _, err := signedSession(); err != nil {
		return nil, err
	}
	if bizType <= 0 {
		bizType = CommentBizSong
	}
	param := map[string]any{"Content": content, "BizType": bizType, "BizId": strconv.FormatInt(bizID, 10)}
	if replyTo != "" {
		param["RepliedCmId"] = replyTo
	}
	if bizSubType > 0 {
		param["BizSubType"] = bizSubType
	}
	var data struct {
		SubCode  int    `json:"SubCode"`
		Message  string `json:"Msg"`
		ID       string `json:"AddedCmId"`
		ParentID string `json:"ParentCmId"`
		Floor    *struct {
			Num int `json:"Num"`
		} `json:"Floor"`
		VerifyURL string `json:"VerifyUrl"`
	}
	if err := rpc("music.globalComment.CommentWriteServer", "AddComment", param, &data); err != nil {
		return nil, err
	}
	if data.SubCode != 0 {
		message := data.Message
		if message == "" {
			message = fmt.Sprintf("SubCode %d", data.SubCode)
		}
		if data.VerifyURL != "" {
			message += " (" + data.VerifyURL + ")"
		}
		return nil, fmt.Errorf("qqmusic: comment rejected: %s", message)
	}
	added := &AddedComment{SubCode: data.SubCode, Message: data.Message, ID: data.ID, ParentID: data.ParentID, VerifyURL: data.VerifyURL}
	if data.Floor != nil {
		added.Floor = data.Floor.Num
	}
	return added, nil
}

// DeleteComment removes one of the account's comments. A missing reply code
// counts as success, as a comment that is already gone does.
func DeleteComment(cmID string) (bool, error) {
	if _, err := signedSession(); err != nil {
		return false, err
	}
	var data struct {
		SubCode *int `json:"SubCode"`
	}
	if err := rpc("music.globalComment.CommentWriteServer", "DelComment", map[string]any{"CommentId": cmID}, &data); err != nil {
		return false, err
	}
	return data.SubCode == nil || *data.SubCode == 0, nil
}

func decodeCommentItems(entries []object) []CommentItem {
	out := make([]CommentItem, 0, len(entries))
	for _, o := range entries {
		id := text(o, "CmId")
		if id == "" {
			continue
		}
		out = append(out, CommentItem{
			ID: id, SeqNo: text(o, "SeqNo"), Nickname: text(o, "Nick"),
			Avatar: text(o, "Avatar"), EncryptUin: text(o, "EncryptUin"),
			Content: html.UnescapeString(text(o, "Content")), Time: number(o, "PubTime"),
			Likes: number(o, "PraiseNum"), Replies: number(o, "ReplyCnt"),
			Praised: boolField(o, "IsPraised"), Self: boolField(o, "IsSelf"),
			State: int(number(o, "State")), HotScore: text(o, "HotScore"),
			RecScore: text(o, "RecScore"), SongID: number(o, "SongId"),
			SongName: text(o, "SongName"), Singers: text(o, "SingerNames"),
			Timeline: rawList(o, "SongTsElems"), HashTags: rawList(o, "HashTagList"),
			Tails: rawList(o, "LittleTails"), Icons: rawList(o, "IconList"),
			VIP: o["VipUI"], SubReplies: rawList(o, "SubComments"),
		})
	}
	return out
}

// boolField reads QQ's boolean fields, which it sends as bools, numbers or strings.
func boolField(o object, keys ...string) bool {
	for _, k := range keys {
		if rawBool(o[k]) {
			return true
		}
	}
	return false
}

func rawBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "1" || s == "true"
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		i, err := strconv.ParseInt(n.String(), 10, 64)
		return err == nil && i != 0
	}
	return false
}

func rawList(o object, keys ...string) []json.RawMessage {
	for _, k := range keys {
		var out []json.RawMessage
		if json.Unmarshal(o[k], &out) == nil && out != nil {
			return out
		}
	}
	return nil
}
