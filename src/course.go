package main

import (
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
}
	
func get_course_info(name string, db redis.Client) CourseInfo {
	aliases, _ := db.Smembers("course:"+name+":aliases")
	return CourseInfo { Name : name, Aliases : aliases }
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

func GetLabsHandler(req Request) Response {
	course := req.param
	if course == "" {
		return Response { code : BadRequest, msg : "Need course name" }
	}

	ids, _ := req.db.Smembers("course:"+course+":labs")
	reply := "["
	for _, id := range ids {
		lab, _ := req.db.Get("course:"+course+":lab:"+id)
		reply += lab + ","
	}
	reply += "]"
	return Response { msg : reply }
}

func is_access_allowed(req Request) bool {
	user := req.user
	course := req.param
	is_admin, _ := req.db.Sismember("admins", user)
	is_my_course, _ := req.db.Sismember("user:"+user+":primary-courses", course)
	is_disabled_marker, _ := req.db.Sismember("course:"+course+":disabled-markers", user)

	if req.ops == "get" {	// Admins and active marker can access /course/get
		return !is_disabled_marker
	} else if req.ops == "get-labs" { // Admins and active markers can access /course/:course/get-labs
		return is_my_course && !is_disabled_marker
	} else if is_admin {	// Only admins can access the rest of the APIs
		return req.ops == "new" || req.ops == "del-alias" || is_my_course
	} else {
		return false
	}
}
