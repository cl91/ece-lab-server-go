function get_parameter_by_name(name) {
    name = name.replace(/[\[]/, "\\[").replace(/[\]]/, "\\]");
    var regex = new RegExp("[\\?&]" + name + "=([^&#]*)"),
    results = regex.exec(location.search);
    return results == null ? "" : decodeURIComponent(results[1].replace(/\+/g, " "));
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
