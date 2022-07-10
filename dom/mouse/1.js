function f(e) {
	console.log(e);
}

var input = document.getElementsByTagName('input')[0];
input.addEventListener('focus', f);
