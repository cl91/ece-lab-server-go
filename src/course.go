package main

import (
	"time"
	"strconv"
	"encoding/json"
	"./redis"
)

func CourseHandler(req Request) Response {
	if !is_access_allowed(req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["new"] = NewCourseHandler
	mux["del"] = DelCourseHandler
	mux["get"] = GetCourseHandler
	mux["new-alias"] = NewAliasHandler
	mux["del-alias"] = DelAliasHandler
	mux["new-marker"] = NewMarkerHandler
	mux["enable-marker"] = EnableMarkerHandler
	mux["disable-marker"] = DisableMarkerHandler
	mux["get-markers"] = GetMarkersHandler
	mux["get-labs"] = GetLabsHandler
	mux["edit-lab"] = EditLabHandler
	mux["update-student-list"] = UpdateStudentListHandler
	mux["get-student-list"] = GetStudentListHandler
	return HandleMux(mux, req.ops, req)
}

func NewCourseHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	name := namev[0]
	r, _ := req.db.Sismember("courses", name)
	if r {
		return Response { code : BadRequest, msg : "Course "+ name + " exists" }
	}
	req.db.Sadd("courses", name)
	req.db.Sadd("user:"+req.user+":primary-courses", name)
	return Response { msg : "Added course " + name }
}

type CourseInfo struct {
	Name string `json:"name"`
	Aliases []string `json:"aliases"`
	Labinfo LabInfo `json:"lab_info"`
}
	
func get_course_info(name string, db redis.Client) CourseInfo {
	aliases, _ := db.Smembers("course:"+name+":aliases")
	return CourseInfo { Name : name, Aliases : aliases,
		Labinfo : get_lab_info(name, db) }
}

func GetCourseHandler(req Request) Response {
	courses, _ := req.db.Smembers("user:"+req.user+":primary-courses")
	obj := make([]CourseInfo, len(courses), len(courses))
	for i, v := range courses {
		obj[i] = get_course_info(v, req.db)
	}
	reply, err := json.Marshal(obj)
	if err != nil {
		return Response { code : ServerError,
			msg : "Error marshalling json objects" }
	}
	return Response { msg : string(reply) }
}

func DelCourseHandler(req Request) Response {
	user := req.user
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	is_course, _ := req.db.Sismember("user:"+user+":primary-courses", course)
	if is_course {
		aliases, _ := req.db.Smembers("course:"+course+":aliases")
		if len(aliases) != 0 {
			return Response { code : BadRequest, msg : "Course " +
				course + " has aliases. Please delete them first." }
		} else {
			req.db.Srem("courses", course)
			req.db.Srem("user:"+user+":primary-courses", course)
			return Response { msg : "Course " + course + " deleted." }
		}
	} else {
		return Response { code : BadRequest, msg : "Invalid course name" }
	}
}

func NewAliasHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need alias course name" }
	}
	name := namev[0]
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	name_exists, _ := req.db.Sismember("courses", name)
	if name_exists {
		return Response { code : BadRequest, msg : "Course " + name + " exists" }
	} else {
		req.db.Sadd("courses", name)
		req.db.Sadd("course:"+course+":aliases", name)
		req.db.Set("course:"+name+":aliased-to", course)
		return Response { msg : "Added alias " + name + " for course " + course }
	}
}

func DelAliasHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need alias course name" }
	}
	name := namev[0]

	course, _ := req.db.Get("course:"+name+":aliased-to")
	if course != "" {
		req.db.Srem("courses", name)
		req.db.Srem("course:"+course+":aliases", name)
		req.db.Del("course:"+name+":aliased-to")
		return Response { msg : "Deleted alias " + name + " for course " + course }
	} else {
		return Response { code : BadRequest,
			msg : "Course " + name + " is not an alias course" }
	}
}

func NewMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	user_exists, _ := req.db.Sismember("users", name)
	if !user_exists {
		req.db.Sadd("users", name)
		req.db.Hset("user:"+name, "pass", name)
		req.db.Hset("user:"+name, "type", "marker")
		req.db.Sadd("user:"+name+":primary-courses", course)
	}

	r, _ := req.db.Sadd("course:"+course+":markers", name)
	if r {
		req.db.Srem("course:"+course+":disabled-markers", name)
		req.db.Hset("user:"+name, "type", "marker")
		return Response { msg : "Added marker " + name + " for course " + course }
	} else {
		return Response { code : ServerError,
			msg : "Failed to add marker " + name + " for course " + course }
	}
}

func DisableMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	is_marker, _ := req.db.Sismember("course:"+course+":markers", name)
	if is_marker {
		req.db.Sadd("course:"+course+":disabled-markers", name)
		req.db.Srem("course:"+course+":markers", name)
		req.db.Srem("user:"+name+":primary-courses", course)
		req.db.Hset("user:"+name, "type", "")
		return Response { msg : "Disabled marker " + name + " for course " + course }
	} else {
		return Response { code : BadRequest,
			msg : "User " + name + " is not a marker for course " + course }
	}
}

func EnableMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	is_disabled, _ := req.db.Sismember("course:"+course+":disabled-markers", name)
	if is_disabled {
		req.db.Srem("course:"+course+":disabled-markers", name)
		req.db.Sadd("course:"+course+":markers", name)
		req.db.Sadd("user:"+name+":primary-courses", course)
		req.db.Hset("user:"+name, "type", "marker")
		return Response { msg : "Enabled marker " + name + " for course " + course }
	} else {
		return Response { code : BadRequest,
			msg : "User " + name + " is not a disabled LOL marker for course " + course }
	}
}

type Markers struct {
	Markers []string `json:"markers"`
	DisabledMarkers []string `json:"disabled"`
}

func GetMarkersHandler(req Request) Response {
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	markers, _ := req.db.Smembers("course:"+course+":markers")
	disabled_markers, _ := req.db.Smembers("course:"+course+":disabled-markers")
	obj := Markers { Markers : markers, DisabledMarkers : disabled_markers }
	reply, err := json.Marshal(obj)
	if err != nil {
		return Response { code : ServerError, msg : "Error marshalling json objects" }
	}
	return Response { msg : string(reply) }
}

func ParseTime(value string) (time.Time, error) {
	loc, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		return time.Time{}, err
	}
	const format = "2006-01-02 15:04"
	return time.ParseInLocation(format, value, loc)
}

type Lab struct {
	Name string `json:"name"`
	Week int `json:"week"`
	MarkingStart string `json:"marking_start"`
	MarkingEnd string `json:"marking_end"`
	MarkingType string `json:"marking"`
	TotalMark int `json:"total_mark"`
	Criteria []MarkingCriterion `json:"criteria"`
}

type MarkingCriterion struct {
	Mark int `json:"mark"`
	Text string `json:"text"`
}

type LabInfo struct {
	Ids []int `json:"ids"`
	ActiveIds []int `json:"active_ids"`
	Labs []Lab `json:"labs"`
}

func max(a []int) (m int) {
	if len(a) == 0 {
		return 0
	}
	m = a[0]
	for _, v := range a {
		if m < v {
			m = v
		}
	}
	return
}

func is_active_lab(lab Lab) bool {
	t1, e1 := ParseTime(lab.MarkingStart)
	t2, e2 := ParseTime(lab.MarkingEnd)
	if e1 != nil || e2 != nil {
		return false
	}
	t := time.Now()
	if t1.Before(t) && t.Before(t2) {
		return true
	} else {
		return false
	}
}

func get_lab_info(course string, db redis.Client) LabInfo {
	ids_str, _ := db.Smembers("course:"+course+":labs")
	ids := make([]int, len(ids_str))
	for i, id := range ids_str {
		parsed, _ := strconv.ParseInt(id, 10, 32)
		ids[i] = int(parsed)
	}
	max_id := max(ids)
	obj := LabInfo { Ids : ids, Labs : make([]Lab, max_id+1) }
	for i, id := range ids {
		lab, _ := db.Get("course:"+course+":lab:"+ids_str[i])
		parsed := Lab {}
		json.Unmarshal([]byte(lab), &parsed)
		obj.Labs[id] = parsed
	}
	active_ids := make([]int, 0)
	i := 0
	for _, id := range ids {
		if is_active_lab(obj.Labs[id]) {
			active_ids = append(active_ids, id)
			i++
		}
	}
	obj.ActiveIds = active_ids
	return obj
}

func GetLabsHandler(req Request) Response {
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	obj := get_lab_info(course, req.db)
	reply, _ := json.Marshal(obj)
	return Response { msg : string(reply) }
}

func EditLabHandler(req Request) Response {
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	idv, ok := req.query["id"]
	if !ok {
		return Response { code : BadRequest, msg : "Need lab id" }
	}
	id := idv[0]
	lab := Lab {}
	json.Unmarshal(req.body, &lab)
	if lab.Week <= 0 {
		return Response { code : BadRequest, msg : "Invalid lab week." }
	}
	if lab.MarkingType == "number" || lab.MarkingType == "attendance" {
		if lab.TotalMark <= 0 {
			return Response { code : BadRequest, msg : "Invalid total mark." }
		}
	} else if lab.MarkingType == "criteria" {
		if lab.Criteria == nil || len(lab.Criteria) == 0 {
			return Response { code : BadRequest, msg : "Invalid criteria." }
		}
		total_mark := 0
		for _, v := range lab.Criteria {
			total_mark += v.Mark
		}
		lab.TotalMark = total_mark
	} else {
		return Response { code : BadRequest, msg : "Invalid marking type." }
	}
	_, err := ParseTime(lab.MarkingStart)
	if err != nil {
		return Response { code : BadRequest, msg : "Invalid marking start time." }
	}
	_, err = ParseTime(lab.MarkingEnd)
	if err != nil {
		return Response { code : BadRequest, msg : "Invalid marking end time." }
	}
	req.db.Sadd("course:"+course+":labs", id)
	stored, _ := json.Marshal(lab)
	req.db.Set("course:"+course+":lab:"+id, string(stored))
	return Response { msg : "Successfully updated lab " + id + " for course " + course }
}

type StudentInfo struct {
	Name string `json:"name"`
	Upi string `json:"upi"`
	Id string `json:"id"`
	Email string `json:"email"`
}

func add_student_to_course(stu StudentInfo, course string, db redis.Client) {
	k := "student:"+stu.Id
	db.Hset(k, "name", stu.Name)
	db.Hset(k, "upi", stu.Upi)
	db.Hset(k, "email", stu.Email)
	db.Sadd(k+":courses", course)
	db.Sadd("course:"+course+":students", stu.Id)
	db.Sadd("students", stu.Id)
	db.Hset("student-upi-to-id", stu.Upi, stu.Id)
	pass, e := db.Hget(k, "pass")
	if e != nil || pass == "" {
		db.Hset(k, "pass", stu.Upi)
	}
}

func get_student_info(id string, db redis.Client) StudentInfo {
	k := "student:" + id
	name, _ := db.Hget(k, "name")
	upi, _ := db.Hget(k, "upi")
	email, _ := db.Hget(k, "email")
	return StudentInfo { Name : name, Upi : upi, Id : id, Email : email }
}

func UpdateStudentListHandler(req Request) Response {
	coursev, ok := req.query["course"]
	if !ok {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	course := coursev[0]
	list := make([]StudentInfo, 0)
	if err := json.Unmarshal(req.body, &list); err != nil {
		return Response { code : BadRequest,
			msg : "Failed to parse json input: " + err.Error() }
	}
	for _, v := range list {
		if v.Name == "" || v.Upi == "" || v.Id == "" {
			continue
		}
		add_student_to_course(v, course, req.db)
	}
	return Response { msg : "Successfully updated student list for course " + course }
}

func GetStudentListHandler(req Request) Response {
	coursev, ok := req.query["course"]
	if !ok {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	course := coursev[0]
	ids, err := req.db.Smembers("course:"+course+":students")
	if err != nil {
		return Response { code : BadRequest,
			msg : "Failed to access student list for course " + course + ":" + err.Error() }
	}
	obj := make([]StudentInfo, len(ids))
	for i, id := range ids {
		obj[i] = get_student_info(id, req.db)
	}
	reply, e := json.Marshal(obj)
	if e != nil {
		return Response { code : ServerError,
			msg : "Error marshalling json object: " + err.Error() }
	}
	return Response { msg : string(reply) }
}

func is_access_allowed(req Request) bool {
	user := req.user
	course := req.param
	is_admin, _ := req.db.Sismember("admins", user)
	is_my_course, _ := req.db.Sismember("user:"+user+":primary-courses", course)
	is_disabled_marker, _ := req.db.Sismember("course:"+course+":disabled-markers", user)

	if req.ops == "get" || req.ops == "get-student-list" {
		// Admins and active markers can access /course/get and /course/get-student-list
		return !is_disabled_marker
	} else if req.ops == "get-labs" {
		// Admins and active markers can access /course/:course/get-labs
		return is_my_course && !is_disabled_marker
	} else if is_admin {
		// Only admins can access the rest of the APIs
		return req.ops == "new" || req.ops == "del-alias" ||
			req.ops == "update-student-list" || is_my_course
	} else {
		return false
	}
}
