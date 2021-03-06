package main

import (
	"encoding/json"
	"./redis"
)

func AdminHandler(req Request) Response {
	if !is_superadmin_user(req) {
		return Response { code : Unauthorised, msg : "Access denied" }
	}
	mux := make(Mux)
	mux["new"] = NewAdminHandler
	mux["passwd"] = AdminPasswdHandler
	mux["del"] = DelAdminHandler
	mux["get"] = GetAdminHandler
	return HandleMux(mux, req.ops, req)
}

func NewAdminHandler(req Request) Response {
	namev, ok1 := req.query["name"];
	passv, ok2 := req.query["pass"];
	fullnamev, ok3 := req.query["fullname"]
	if (!ok1 || !ok2) {
		return Response { code : BadRequest, msg : "Need admin name and password" }
	}
	name := namev[0]
	pass := passv[0]
	fullname := ""
	if ok3 {
		fullname = fullnamev[0]
	}

	user_exists, _ := req.db.Sismember("users", name)
	if user_exists {
		return Response { code : BadRequest, msg : "User " + name + " exists" }
	}
	
	req.db.Sadd("users", name)
	req.db.Hset("user:"+name, "pass", pass)
	req.db.Hset("user:"+name, "type", "admin")
	if fullname != "" {
		req.db.Hset("user:"+name, "fullname", fullname)
	}

	r, _ := req.db.Sadd("admins", name)
	if r {
		return Response { msg : "Admin " + name + " added" }
	} else {
		return Response { code : ServerError, msg : "Failed to add admin " + name }
	}
}

func AdminPasswdHandler(req Request) Response {
	namev, ok1 := req.query["name"];
	passv, ok2 := req.query["pass"];
	if (!ok1 || !ok2) {
		return Response { code : BadRequest, msg : "Need admin name and new password" }
	}
	name := namev[0]
	pass := passv[0]

	user_exists, _ := req.db.Sismember("users", name)
	if !user_exists {
		return Response { code : BadRequest, msg : "User " + name + " does not exist" }
	}
	
	req.db.Hset("user:"+name, "pass", pass)
	return Response { msg : "Updated password for admin " + name }
}

func DelAdminHandler(req Request) Response {
	namev, ok := req.query["name"]
	if !ok {
		return Response { code : BadRequest, msg : "Need admin name" }
	}
	name := namev[0]
	r, _ := req.db.Sismember("admins", name)
	if !r {
		return Response { code : BadRequest,
			msg : "Admin " + name + " does not exist" }
	}
	req.db.Srem("admins", name)
	req.db.Srem("users", name)
	req.db.Del("user:"+name)
	req.db.Del("user:"+name+":courses")
	return Response { msg : "Admin " + name + " deleted" }
}

type AdminInfo struct {
	Name string `json:"name"`
	Fullname string `json:"fullname"`
	Courses []CourseInfo `json:"courses"`
}

func get_admin_info(admin string, db redis.Client) (AdminInfo, error) {
	courses, err := get_all_courses(admin, db, false)
	if err != nil {
		return AdminInfo {}, err
	}
	fullname, _ := db.Hget("user:"+admin, "fullname")
	if fullname == "" {
		fullname = admin
	}
	return AdminInfo { Name:admin, Fullname:fullname,
		Courses:courses }, nil
}

func GetAdminHandler(req Request) Response {
	admins, err := req.db.Smembers("admins")
	if err != nil {
		return Response { code : ServerError,
			msg : "Db access failed: " + err.Error() }
	}
	info := make([]AdminInfo, len(admins), len(admins))
	for i, name := range admins {
		info[i], _ = get_admin_info(name, req.db)
	}
	reply, err1 := json.Marshal(info)
	if err1 != nil {
		return Response { code : ServerError,
			msg : "Error marshaling json object: " + err1.Error() }
	}
	return Response { msg: string(reply) }
}

func is_superadmin_user(req Request) bool {
	reply, _ := req.db.Sismember("superadmins", req.user)
	return reply
}
