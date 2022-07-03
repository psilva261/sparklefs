// TODO:
// pure dom click
// document.getElementById('ui-id-4').parentElement.click()

var thisId, targetId, curId;

var b = document.getElementById('b');
var p = document.getElementById('p');
function print() {
	console.log('this: ' + thisId);
	console.log('e.target: ' + targetId);
	console.log('e.currentTarget: ' + curId);
}
function f(e) {
	console.log('')
	console.log('======= f(..) =======')
	console.log('this:', this);
	thisId = this.id;
	targetId = e.target.id;
	curId = e.currentTarget.id;
	print();
}
b.addEventListener('click', f);
p.addEventListener('click', f);
document.addEventListener('click', f);
window.addEventListener('click', f);
console.log('dispatch on b');
e = new Event('click', {bubbles: true});
b.dispatchEvent(e);
console.log('dispatch on p');
e = new Event('click', {bubbles: true});
p.dispatchEvent(e);
console.log('dispatch on doc');
e = new Event('click', {bubbles: true});
document.dispatchEvent(e);
console.log('click on p:');
p.click();