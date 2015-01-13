package main

import (
	"flag"
	"./redis"
)

type Handler func(Request) Response
type Mux map[string] Handler

func HandleMux(mux Mux, key string, req Request) Response {
	f, ok := mux[key]
	if ok {
		return f(req)
	} else {
		return Response { code : BadRequest, msg : "Invalid api request." }
	}
}

func HandleApi(req Request) Response {
	db, err := InitDb()
	if err != nil {
		return Response { code : ServerError, msg : "Failed to connect to database " + err.Error() }
	}
	defer db.Quit()
	req.db = db

	req.user = ParseUser(req)
	req.primary_course = ParsePrimaryCourse(req)

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

func ParseUser(req Request) string {
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
			return user
		}
	}
	return ""
}

func ParsePrimaryCourse(req Request) string {
	aliased_to, _ := req.db.Get("course:"+course+":aliased-to")
	primary_course := req.course
	if aliased_to != "" {
		primary_course = aliased_to
	}
	return primary_course
}
