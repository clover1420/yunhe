package qqmusic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Comment counts, thread pages and writes decode QQ's replies and send the
// field names its pages spell.
func TestCommentExtras(t *testing.T) {
	params := make(map[string]map[string]any)
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
		params[request.Req.Method] = request.Req.Param
		switch request.Req.Method {
		case "GetCmCount":
			rpcReply(w, map[string]any{
				"response": map[string]any{
					"biz_type": 1, "biz_id": "42", "biz_sub_type": 2,
					"count": 1234, "count_ver": "v1", "count_view": "1234",
					"related_id": "r1", "tip": "tip",
					"icon_list": []any{map[string]any{
						"txt": "热", "unique_id": "u1", "type": 1, "cmid": "c9", "is_dynamic": true,
					}},
				},
				"cmTabType": 3,
			})
		case "GetHotCommentList":
			rpcReply(w, map[string]any{
				"CommentList": map[string]any{
					"Comments": []any{
						map[string]any{"CmId": "11", "SeqNo": "seq1", "Nick": "n", "Avatar": "a", "Content": "hello &amp; bye", "PubTime": 100, "PraiseNum": 5, "ReplyCnt": 2, "IsSelf": 1, "IsPraised": 1},
						map[string]any{"CmId": "12", "SeqNo": "seq2", "Nick": "n2", "Content": "second"},
					},
					"HasMore": 1, "Total": 9,
				},
				"TotalCmNum": 9, "CommentTip": "tip", "HasTsCm": 1, "ShareCnt": 4, "SubCode": 0,
			})
		case "GetSongTsCmList":
			rpcReply(w, map[string]any{
				"CmList": []any{map[string]any{
					"CmId": "21", "SeqNo": "s21", "Content": "moment", "PubTime": 30,
					"PraiseNum": 1, "ReplyCnt": 0, "IsSelf": 0, "Location": "上海",
				}},
				"HasMore": 0, "NextPos": "p2", "Hint": "hint", "PrevListLoaded": 1,
			})
		case "AddComment":
			if request.Req.Param["Content"] == "hello" {
				rpcReply(w, map[string]any{"SubCode": 0, "Msg": "", "AddedCmId": "c1", "ParentCmId": "", "Floor": map[string]any{"Num": 2}})
			} else {
				rpcReply(w, map[string]any{"SubCode": 10002, "Msg": "need captcha", "VerifyUrl": "https://verify"})
			}
		case "DelComment":
			rpcReply(w, map[string]any{"SubCode": 0})
		default:
			t.Errorf("unexpected RPC %s/%s", request.Req.Module, request.Req.Method)
		}
	})
	setSession(Session{MusicID: "10001", MusicKey: "fake", LoginType: 2})

	count, err := CommentCount(42, CommentBizSong, 0)
	if err != nil || count.Count != 1234 || count.BizID != "42" || count.TabType != 3 || len(count.Icons) != 1 || !count.Icons[0].Dynamic {
		t.Fatalf("CommentCount = %+v, %v", count, err)
	}
	request, _ := params["GetCmCount"]["request"].(map[string]any)
	if request["biz_id"] != "42" || request["biz_type"] != float64(1) || request["biz_sub_type"] != float64(2) {
		t.Errorf("CommentCount sent %v", request)
	}

	page, err := HotComments(42, 1, 15, "", CommentBizSong, 0)
	if err != nil || len(page.Comments) != 2 || page.Comments[0].Content != "hello & bye" || page.NextSeqNo != "seq2" || !page.HasMore || page.Total != 9 {
		t.Fatalf("HotComments = %+v, %v", page, err)
	}
	if _, ok := params["GetHotCommentList"]["BizSubType"]; ok {
		t.Error("HotComments sent BizSubType although none was given")
	}
	if params["GetHotCommentList"]["PageNum"] != float64(0) {
		t.Errorf("HotComments sent PageNum %v, want 0", params["GetHotCommentList"]["PageNum"])
	}
	if _, err = HotComments(42, 2, 15, page.NextSeqNo, CommentBizSong, 0); err != nil {
		t.Fatal(err)
	}
	if params["GetHotCommentList"]["PageNum"] != float64(1) || params["GetHotCommentList"]["LastCommentSeqNo"] != "seq2" {
		t.Errorf("HotComments cursor = %v", params["GetHotCommentList"])
	}

	moment, err := MomentComments(42, 15, "", CommentBizSong, 0)
	if err != nil || len(moment.Comments) != 1 || moment.Comments[0].ID != "21" || moment.NextPos != "p2" || moment.HasMore {
		t.Fatalf("MomentComments = %+v, %v", moment, err)
	}
	if params["GetSongTsCmList"]["LastPos"] != "" || params["GetSongTsCmList"]["SeekTs"] != float64(-1) {
		t.Errorf("MomentComments sent %v", params["GetSongTsCmList"])
	}

	added, err := AddComment(42, "hello", "c0", CommentBizSong, 0)
	if err != nil || added.ID != "c1" || added.Floor != 2 {
		t.Fatalf("AddComment = %+v, %v", added, err)
	}
	if params["AddComment"]["RepliedCmId"] != "c0" || params["AddComment"]["BizId"] != "42" {
		t.Errorf("AddComment sent %v", params["AddComment"])
	}
	if _, err = AddComment(42, "spam", "", CommentBizSong, 0); err == nil || !strings.Contains(err.Error(), "https://verify") {
		t.Fatalf("AddComment ignored SubCode: %v", err)
	}
	if ok, err := DeleteComment("c1"); err != nil || !ok {
		t.Fatalf("DeleteComment = %v, %v", ok, err)
	}
}
