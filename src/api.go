package main

import (
	"flag"
	"./redis"
)

type UserType int
const (
	Student UserType = iota
	NonStudent
	InvalidUser
)

func HandleApi(req Request) Response {
	db, err := InitDb()
	if err != nil {
		return Response { code : ServerError, msg : "Failed to connect to database " + err.Error() }
	}
	defer db.Quit()
	req.db = db

	user, utype := ParseUser(req)
	switch utype {
	case Student:
		req.student = user
		req.user = ""
	case NonStudent:
		req.student = ""
		req.user = user
	case InvalidUser:
		req.student = ""
		req.user = ""
	}

	mux := make(Mux)
	mux["auth"] = AuthHandler
	mux["passwd"] = PasswdHandler
	mux["admin"] = AdminHandler
	mux["course"] = CourseHandler
	mux["mark"] = MarkHandler
	mux["student"] = StudentHandler
	return HandleMux(mux, req.api, req)
}

func InitDb() (redis.Client, error) {
	// Parse command-line flags; needed to let flags used by Go-Redis be parsed.
        flag.Parse()

        // create the client.  Here we are using a synchronous client.
        // Using the default ConnectionSpec, we are specifying the client to connect
        // to db 13 (e.g. SELECT 13), and a password of go-redis (e.g. AUTH go-redis)
        spec := redis.DefaultSpec().Db(0);
        return redis.NewSynchClientWithSpec(spec)
}

func ParseUser(req Request) (string, UserType) {
	var auth string
	if req.cookies != nil {
		for _, cookie := range req.cookies {
			if cookie.Name == "auth" {
				auth = cookie.Value
				break
			}
		}
	}
	if auth == "" {
		value, ok := req.query["auth"]
		if ok {
			auth = value[0]
		}
	}
	if auth != "" {
		user, _ := req.db.Hget("auth", auth)
		user_auth, _ := req.db.Hget("user:"+user, "auth")
		if auth == user_auth {
			return user, NonStudent
		}
		stu_id, _ := req.db.Hget("student-auth", auth)
		stu_auth, _ := req.db.Hget("student:"+stu_id, "auth")
		if auth == stu_auth {
			return user, Student
		}
	}
	return "", InvalidUser
}
