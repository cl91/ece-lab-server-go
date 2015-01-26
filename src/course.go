package main

import (
	"time"
	"strconv"
	"encoding/json"
	"./redis"
	"./mapset"
)

func CourseHandler(req Request) Response {
	if !is_access_allowed(&req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["new"] = NewCourseHandler
	mux["del"] = DelCourseHandler
	mux["get"] = GetCourseHandler
	mux["new-alias"] = NewAliasHandler
	mux["new-marker"] = NewMarkerHandler
	mux["enable-marker"] = EnableMarkerHandler
	mux["disable-marker"] = DisableMarkerHandler
	mux["reset-marker-passwd"] = ResetMarkerPasswdHandler
	mux["get-markers"] = GetMarkersHandler
	mux["get-labs"] = GetLabsHandler
	mux["edit-lab"] = EditLabHandler
	mux["update-student-list"] = UpdateStudentListHandler
	mux["get-student-list"] = GetStudentListHandler
	return HandleMux(mux, req.ops, req)
}

// POST /course/new?name=compsys723
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

func get_aliases(name string, db redis.Client) ([]string, error) {
	return db.Smembers("course:"+name+":aliases")
}

func get_course_info(name string, db redis.Client) CourseInfo {
	aliases, _ := get_aliases(name, db)
	return CourseInfo { Name : name, Aliases : aliases,
		Labinfo : get_lab_info(name, db) }
}

func get_all_courses(user string, db redis.Client, is_student bool) ([]CourseInfo, error) {
	prefix := "user:"
	if is_student {
		prefix = "student:"
	}
	courses, err := db.Smembers(prefix+user+":primary-courses")
	if err != nil {
		return nil, err
	}
	obj := make([]CourseInfo, len(courses), len(courses))
	for i, v := range courses {
		obj[i] = get_course_info(v, db)
	}
	return obj, nil
}

// POST /course/get
func GetCourseHandler(req Request) Response {
	courses, err := get_all_courses(req.get_user(), req.db, (req.user == ""))
	if err != nil {
		return Response { code : BadRequest,
			msg : "Error getting course information" }
	}
	reply, err := json.Marshal(courses)
	if err != nil {
		return Response { code : ServerError,
			msg : "Error marshalling json objects" }
	}
	return Response { msg : string(reply) }
}

// /course/:course/del
func DelCourseHandler(req Request) Response {
	user := req.user
	course := req.course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	is_primary_course := req.primary_course == req.course
	if is_primary_course {
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
		req.db.Srem("courses", course)
		req.db.Srem("course:" + req.primary_course + ":aliases", course)
		req.db.Del("course:" + course + ":aliased-to")
		return Response { msg : "Deleted alias " + course +
			" for course " + req.primary_course }
	}
}

// POST /course/:course/new-alias?name=mecheng701
func NewAliasHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need alias course name" }
	}
	name := namev[0]
	course := req.course
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

// POST /course/:course/new-marker?name=odep012
func NewMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	passv, ok := req.query["pass"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker password" }
	}
	pass := passv[0]
	course := req.primary_course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	user_exists, _ := req.db.Sismember("users", name)
	if user_exists {
		return Response { code : BadRequest, msg : "User " + name + " exists." }
	}

	req.db.Sadd("users", name)
	req.db.Hset("user:"+name, "pass", pass)
	req.db.Hset("user:"+name, "type", "marker")
	req.db.Sadd("user:"+name+":primary-courses", course)
	req.db.Sadd("course:"+course+":markers", name)
	req.db.Srem("course:"+course+":disabled-markers", name)
	return Response { msg : "Added marker " + name + " for course " + course }
}

// POST /course/:course/disable-marker?name=odep012
func DisableMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	course := req.course
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

// POST /course/:course/enable-marker?name=odep012
func EnableMarkerHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	course := req.course
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

// POST /course/:course/reset-marker-passwd?name=abc&pass=abc
func ResetMarkerPasswdHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need marker name" }
	}
	name := namev[0]
	passv, ok := req.query["pass"]
	if !ok {
		return Response { code : BadRequest, msg : "Need new password" }
	}
	pass := passv[0]
	course := req.course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	is_marker, _ := req.db.Sismember("course:"+course+":markers", name)
	is_disabled_marker, _ := req.db.Sismember("course:"+course+":disabled-markers", name)
	if is_marker || is_disabled_marker {
		req.db.Hset("user:"+name, "pass", pass)
		return Response { msg : "Resetted password for marker " + name + " of course " + course }
	} else {
		return Response { code : BadRequest,
			msg : "User " + name + " is not a marker for course " + course }
	}
}

type Markers struct {
	Markers []string `json:"markers"`
	DisabledMarkers []string `json:"disabled"`
}

// POST /course/:course/get-markers
func GetMarkersHandler(req Request) Response {
	course := req.course
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

// POST /course/:course/get-labs
func GetLabsHandler(req Request) Response {
	course := req.primary_course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	obj := get_lab_info(course, req.db)
	reply, _ := json.Marshal(obj)
	return Response { msg : string(reply) }
}

// POST /course/:course/edit-lab?id=1
// DATA see Lab struct
func EditLabHandler(req Request) Response {
	course := req.course
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
	Marked bool `json:"marked"`
	TotalMark uint `json:"total_mark"`
}

func add_student_to_course(stu StudentInfo, course string, primary_course string, db redis.Client) {
	k := "student:"+stu.Id
	db.Hset(k, "name", stu.Name)
	db.Hset(k, "upi", stu.Upi)
	db.Hset(k, "email", stu.Email)
	db.Sadd(k+":courses", course)
	db.Sadd(k+":primary-courses", primary_course)
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

func get_student_mark_for_lab(id string, course string, lab string, db redis.Client) (uint, bool) {
	lab_id, err := strconv.ParseUint(lab, 10, 32)
	if err != nil {
		return 0, false
	}
	markv, err := get_marks(course, lab_id, id, db)
	if err != nil {
		return 0, false
	}
	var total_mark uint
	if len(markv) > 0 && len(markv[0].Mark) > 0 {
		for _, v := range markv[0].Mark {
			total_mark += v
		}
		return total_mark, true
	} else {
		return 0, false
	}
}

func get_student_ids(course string, merge bool, db redis.Client) (ids []string, err error) {
	if !merge {
		return db.Smembers("course:"+course+":students")
	} else {
		aliases, err := get_aliases(course, db)
		if err != nil {
			return nil, err
		}
		prim_course_stu_ids, err := get_student_ids(course, false, db)
		if err != nil {
			return nil, err
		}
		// Is Rob Pike this retarded? WTF?!! No generics?!!!
		prim_course_stu_ids_conv := make([]interface{}, len(prim_course_stu_ids))
		for i, v := range prim_course_stu_ids {
			prim_course_stu_ids_conv[i] = interface{}(v)
		}
		all_stu_ids := mapset.NewSetFromSlice(prim_course_stu_ids_conv)
		for _, alias := range aliases {
			alias_stu_ids, err := get_student_ids(alias, false, db)
			if err != nil {
				return nil, err
			}
			alias_stu_ids_conv := make([]interface{}, len(alias_stu_ids))
			for i, v := range alias_stu_ids {
				alias_stu_ids_conv[i] = interface{}(v)
			}
			all_stu_ids = all_stu_ids.Union(
				mapset.NewSetFromSlice(alias_stu_ids_conv))
		}
		all_stu_ids_ifce := all_stu_ids.ToSlice()
		all_stu_ids_str := make([]string, len(all_stu_ids_ifce))
		for i, v := range all_stu_ids_ifce {
			all_stu_ids_str[i] = v.(string)
		}
		return all_stu_ids_str, nil
	}
}

// POST /course/:course/update-student-list
// DATA array of StudentInfo
func UpdateStudentListHandler(req Request) Response {
	course := req.course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	list := make([]StudentInfo, 0)
	if err := json.Unmarshal(req.body, &list); err != nil {
		return Response { code : BadRequest,
			msg : "Failed to parse json input: " + err.Error() }
	}
	for _, v := range list {
		if v.Name == "" || v.Upi == "" || v.Id == "" {
			continue
		}
		add_student_to_course(v, course, req.primary_course, req.db)
	}
	return Response { msg : "Successfully updated student list for course " + course }
}

// POST /course/:course/get-student-list[?lab=1][&merge=true]
func GetStudentListHandler(req Request) Response {
	course := req.course
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}
	merge := false
	mergev, ok := req.query["merge"]
	if ok && mergev[0] == "true" {
		merge = true
	}
	if merge {
		course = req.primary_course
	}
	ids, err := get_student_ids(course, merge, req.db)
	if err != nil {
		return Response { code : BadRequest,
			msg : "Failed to access student list for course " + course + ":" + err.Error() }
	}
	lab := ""
	labv, ok := req.query["lab"]
	if ok {
		lab = labv[0]
	}
	obj := make([]StudentInfo, len(ids))
	for i, id := range ids {
		obj[i] = get_student_info(id, req.db)
		if lab != "" {
			mark, marked := get_student_mark_for_lab(id, course, lab, req.db)
			obj[i].Marked = marked
			obj[i].TotalMark = mark
		}
	}
	reply, e := json.Marshal(obj)
	if e != nil {
		return Response { code : ServerError,
			msg : "Error marshalling json object: " + err.Error() }
	}
	return Response { msg : string(reply) }
}

func is_access_allowed(req *Request) bool {
	user := req.user
	course := req.course
	is_admin, _ := req.db.Sismember("admins", user)
	is_my_course, _ := req.db.Sismember("user:"+user+":primary-courses", req.primary_course)
	is_disabled_marker, _ := req.db.Sismember("course:"+course+":disabled-markers", user)

	if req.ops == "get" {
		// Admins, markers, and students can access /course/get
		return true
	} else if req.ops == "get-labs" || req.ops == "get-student-list" {
		// Admins and active markers can access
		// /course/:course/get-labs and /course/:course/get-student-list
		return is_my_course && !is_disabled_marker
	} else if is_admin {
		// Only admins can access the rest of the APIs
		return req.ops == "new" || req.ops == "del-alias" || is_my_course
	} else {
		return false
	}
}
