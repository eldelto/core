const keydownEvent = new Event("keydown");

document.addEventListener("DOMContentLoaded", function() {
	document.querySelectorAll("textarea.auto-grow")
		.forEach(e => {
			e.addEventListener("keydown", autosize);
			e.dispatchEvent(keydownEvent);
		});
});

function autosize(){
	var el = this;
	setTimeout(function(){
		el.style.cssText = 'height:auto; padding:0';
		// for box-sizing other than "content-box" use:
		// el.style.cssText = '-moz-box-sizing:content-box';
		el.style.cssText = 'height:' + el.scrollHeight + 'px';
	},0);
}
