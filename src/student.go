package main

import (
	"os"
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"./redis"
)

func StudentHandler(req Request) Response {
	if !is_student_of_this_course(req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["get-history"] = GetHistoryHandler
	mux["upload"] = UploadCodeHandler
	return HandleMux(mux, req.ops, req)
}

type UploadFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type GetHistoryReply struct {
	Markv []Mark `json:"mark"`
	Uploaded bool `json:"uploaded"`
}

// GET /student/:course/get-history?id=lab_id
func GetHistoryHandler(req Request) Response {
	idv, ok := req.query["id"]
	if !ok {
		return Response { code : BadRequest, msg : "Need lab id" }
	}
	lab_id, _ := strconv.ParseUint(idv[0], 10, 32)
	markv, _ := get_marks(req.course, lab_id, req.student, req.db)
	uploaded_int, _ := req.db.Get(get_key_for_uploaded(req.course, idv[0], req.student))
	uploaded := false
	if uploaded_int == "true" {
		uploaded = true
	}
	obj := GetHistoryReply { Markv : markv, Uploaded : uploaded }
	reply, _ := json.Marshal(obj)
	return Response { msg : string(reply) }
}

func get_student_upi(student string, db redis.Client) string {
	return get_student_info(student, db).Upi
}

// POST /student/:course/upload?id=lab_id
// DATA file in multipart form
func UploadCodeHandler(req Request) Response {
	idv, ok := req.query["id"]
	if !ok {
		return Response { code : BadRequest, msg : "Need lab id" }
	}
	lab_id := idv[0]
	list := make([]UploadFile, 0)
	if err := json.Unmarshal(req.body, &list); err != nil || len(list) == 0 {
		return Response { code : BadRequest,
			msg : "Failed to parse json input: " + err.Error() }
	}

	// open destination
	dir := "./uploaded/" + req.course + "/" + lab_id + "/" +
		get_student_upi(req.student, req.db) + "/"
	err := os.MkdirAll(dir, os.ModeDir | 0755)
	if err != nil {
		return Response { code : ServerError,
			msg : "Failed to create directory: " + err.Error() }
	}
	outfile, err := os.Create(dir + list[0].Name)
	if err != nil {
		return Response { code : ServerError,
			msg : "Failed to open outfile: " + err.Error() }
	}
	// 32K buffer copy
	_, err = io.Copy(outfile, bytes.NewReader(list[0].Data))
	if err != nil {
		return Response { code : ServerError,
			msg : "Failed to write buffer: " + err.Error() }
	}
	req.db.Set(get_key_for_uploaded(req.course, lab_id, req.student), "true")
	return Response { msg : "Uploaded: " + list[0].Name }
}

func get_key_for_uploaded(course string, lab_id string, stu string) string {
	return "student:" + stu + ":course:" + course+":lab:" + lab_id + ":uploaded"
}

func is_student_of_this_course(req Request) bool {
	student := req.student
	course := req.course
	is_my_course, _ := req.db.Sismember("student:"+student+":courses", course)

	return is_my_course
}
