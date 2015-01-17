package main

import (
	"time"
	"strconv"
	"encoding/json"
	"./redis"
)

func MarkHandler(req Request) Response {
	if !is_marker_of_this_course(req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["upload"] = UploadMarkHandler
	mux["get-marks"] = GetMarksHandler
	return HandleMux(mux, req.ops, req)
}

// POST /mark/:course/upload?lab=2&uid=6351823
// Data: '[5]', '[2, 2, 2, 2, 2]'
func UploadMarkHandler(req Request) Response {
	course := req.course
	labv, ok := req.query["lab"]
	if !ok {
		return Response { code : BadRequest, msg : "Need lab id" }
	}
	id, err := strconv.ParseUint(labv[0], 10, 32)
	if err != nil {
		return Response { code : BadRequest, msg : "Invalid lab id" }
	}
	uidv, ok := req.query["uid"]
	if !ok {
		return Response { code : BadRequest, msg : "Need user id" }
	}
	uid := uidv[0]
	markv := make([]uint, 0)
	err = json.Unmarshal([]byte(req.body), &markv)
	if err != nil {
		return Response { code : BadRequest, msg : "Invalid json object" }
	}
	lab_info := get_lab_info(course, req.db)
	lab := lab_info.Labs[id]
	if ((lab.MarkingType == "number" || lab.MarkingType == "attendance") && len(markv) == 1) ||
		(lab.MarkingType == "criteria" && len(markv) == len(lab.Criteria)) {
		update_mark(course, id, uid, markv, req)
		return Response { msg : "Marked student " + uid }
	} else {
		return Response { code : BadRequest, msg : "Marking type mismatch" }
	}
}

// POST /mark/:course/get-marks?lab=2&uid=6351823
func GetMarksHandler(req Request) Response {
	course := req.course
	labv, ok := req.query["lab"]
	if !ok {
		return Response { code : BadRequest, msg : "Need lab id" }
	}
	id, err := strconv.ParseUint(labv[0], 10, 32)
	if err != nil {
		return Response { code : BadRequest, msg : "Invalid lab id" }
	}
	uidv, ok := req.query["uid"]
	if !ok {
		return Response { code : BadRequest, msg : "Need user id" }
	}
	uid := uidv[0]
	markv, err := get_marks(course, id, uid, req.db)
	if err != nil {
		return Response { code : ServerError,
			msg : "Failed to get marks: " + err.Error() }
	}
	reply, _ := json.Marshal(markv)
	return Response { msg : string(reply) }
}

type Mark struct {
	Mark []uint `json:"mark"`
	Date string `json:"date"`
	Marker string `json:"marker"`
}

func update_mark(course string, id uint64, uid string, markv []uint, req Request) {
	const layout = "Monday, January 2, 2006 at 3:04pm"
	mark := Mark { Mark : markv, Date : time.Now().Format(layout),
		Marker : req.user }
	mark_json, _ := json.Marshal(mark)
	req.db.Lpush("student:" + uid + ":course:" + course + ":lab:" +
		strconv.FormatUint(id, 10), string(mark_json))
}

func get_marks(course string, id uint64, uid string, db redis.Client) (markv []Mark, err error) {
	mark_jsonv, err := db.Lrange("student:"+uid+":course:"+course+":lab:"+
		strconv.FormatUint(id, 10), 0, -1)
	if err != nil {
		return nil, err
	}
	markv = make([]Mark, len(mark_jsonv))
	for i, m := range mark_jsonv {
		mark := Mark {}
		_ = json.Unmarshal([]byte(m), &mark)
		markv[i] = mark
	}
	return markv, nil
}

func is_marker_of_this_course(req Request) bool {
	user := req.user
	course := req.primary_course
	is_my_course, _ := req.db.Sismember("user:"+user+":primary-courses", course)
	is_disabled_marker, _ := req.db.Sismember("course:"+course+":disabled-markers", user)

	return is_my_course && !is_disabled_marker
}
