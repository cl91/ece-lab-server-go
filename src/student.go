package main

import (
	"os"
	"bytes"
	"encoding/json"
	"io"
)

func StudentHandler(req Request) Response {
	if !is_student_of_this_course(req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["upload"] = UploadCodeHandler
	return HandleMux(mux, req.ops, req)
}

type UploadFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data []byte `json:"data"`
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
	dir := "./uploaded/" + req.course + "/" + lab_id + "/" + req.student + "/"
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
	return Response { msg : "Uploaded: " + list[0].Name }
}

func is_student_of_this_course(req Request) bool {
	student := req.student
	course := req.course
	is_my_course, _ := req.db.Sismember("student:"+student+":courses", course)

	return is_my_course
}
