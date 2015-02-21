function get_parameter_by_name(name) {
    name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
    var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
    results = regex.exec(location.search);
    return results == null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "));
}

function get_course_fullname(course) {
    var name = course.name
    if (course.aliases) {
	for (var i = 0; i < course.aliases.length; i++) {
	    name += "/" + course.aliases[i]
	}
    }
    return name
}

function get_max_array(a) {
    return Math.max.apply(null, a);
}

function get_template(id) {
    return document.getElementById(id).firstChild.textContent
}

function is_active_marking(ids, id) {
    if (ids.length) {
	for (var i = 0; i < ids.length; i++) {
	    if (ids[i] == id) {
		return "Yes"
	    }
	}
    }
    return "No"
}

function get_textual_marks(marks, lab) {
    var ret = []
    for (var i = 0; i < marks.length; i++) {
	mark = marks[i].mark
	if (!mark) {
	    continue
	}
	texts = []
	if (lab.mtype == 'criteria') {
	    for (var j = 0; j < mark.length; j++) {
		var symb = mark[j] == 0 ? '&#10007;' : '&#10004;'
		texts[j] = symb + ' ' + lab.criteria[j].text
	    }
	} else if (lab.mtype == 'number') {
	    texts[0] = String(mark[0])
	} else if (lab.mtype == 'attendance') {
	    var symb = mark[0] == 0 ? '&#10007;' : '&#10004;'
	    texts[0] = symb + ' Student attended lab.'
	}
	ret[i] = { texts : texts, date : marks[i].date, marker : marks[i].marker,
		     "class" : i == 0 ? "success" : "warning" }
    }
    return ret
}

String.prototype.addSlashes = function() 
{ 
   //no need to do (str+'') anymore because 'this' can only be a string
   return this.replace(/[\\"']/g, '\\$&').replace(/\u0000/g, '\\0');
}
