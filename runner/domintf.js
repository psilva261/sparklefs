window.history = {
	replaceState: function() {}
};
window.screen = {
	width: 1280,
	height: 1024
};
window.screenX = 0;
window.screenY = 25;
window.getComputedStyle = function(el, pseudo) {
	this.el = el;
	this.getPropertyValue = function(prop) {
		return opossum.style(___path('', el), pseudo, prop, arguments[2]);
	};
	return this;
};

btoa = opossum.btoa;

location = window.location;
navigator = {
	platform: 'plan9(port)',
	userAgent: 'opossum'
};

___opossumSubmit = function(a, b, c) {
	if (this.onsubmit) this.onsubmit()
	console.log('submit()');
	if (this.tagName === 'BUTTON' || this.tagName === 'INPUT') {
		let p;
		for (p = el; p = p.parentElement; p != null) {
			if (p.tagName === 'FORM') {
				if (p.onsubmit) p.onsubmit()
				break;
			}
		}
	}
}

getMethods = (obj) => {
  let properties = new Set()
  let currentObj = obj
  do {
    Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
  } while ((currentObj = Object.getPrototypeOf(currentObj)))
  return [...properties.keys()].filter(item => typeof obj[item] === 'function')
}

getProperties = (obj) => {
  let properties = new Set()
  let currentObj = obj
  do {
    Object.getOwnPropertyNames(currentObj).map(item => properties.add(item))
  } while ((currentObj = Object.getPrototypeOf(currentObj)))
  return [...properties.keys()].filter(item => typeof obj[item] !== 'function')
}

function XMLHttpRequest() {
	var _method, _uri;
	var h = {};
	var ls = {};

	this.readyState = 0;

	var cb = function(data, err) {
		if (data !== '') {
			this.responseText = data;
			this.readyState = 4;
			this.state = 200;
			this.status = 200;
			if (ls['load']) ls['load'].bind(this)();
			if (this.onload) this.onload.bind(this)();

			if (this.onreadystatechange) this.onreadystatechange.bind(this)();
		}
	}.bind(this);

	this.addEventListener = function(k, fn) {
		ls[k] = fn;
	};
	this.open = function(method, uri) {
		_method = method;
		_uri = uri;
	};
	this.setRequestHeader = function(k, v) {
		h[k] = v;
	};
	this.send = function(data) {
		opossum.xhr(_method, _uri, h, data, cb);
		this.readyState = 2;
	};
	this.getAllResponseHeaders = function() {
		return '';
	};
}

//
// Fetch API
//

function Headers(hs) {
	let h = {};

	this.append = function(k, v) {
		h[k] = v;
	}
	this.set = function(k, v) {
		h[k] = v;
	}
	this.has = function(k) {
		return !!h[k];
	}
	this.get = function(k) {
		return h[k];
	}
	this.keys = function() {
		return Object.keys(h);
	}
}

function Request(url, opts) {
	this.url = url;
	if (opts) {
		for (var k in opts) {
			this[k] = opts[k];
		}
	}
}

function Response(body) {
  let data = body;

  this.body = function() {
    return data;
  }
  this.json = function() {
    return JSON.parse(data);
  }
  this.text = function() {
    return data;
  }
}

function fetch(resource, init) {
  let url;
  let opts;
  let hs = {};
  if (typeof resource === 'string') {
    url = resource;
    opts = init || {};
  } else {
  	url = resource.url;
  	opts = resource.opts || {};
  	if (resource.headers) {
  		for (var k in resource.headers.keys()) {
  			hs[k] = resource.headers.get(k);
  		}
  	}
  }
  let body = opts['body'] || '';
  let method = opts['method'] || 'GET';
	return new Promise(function(resolve, reject) {
		let cb = function(data, err) {
			if (err) {
				reject(err);
			} else {
				resolve(new Response(data));
			}
		};
		opossum.xhr(method, url, hs, body, cb);
	});
}

var ___path;
___path = function(pre, el) {
	var i, p;

	if (!el) {
		return undefined;
	}
	if (el.tagName === 'BODY') {
		return '/0';
	}
	p = el.parentElement;

	if (p) {
		for (i = 0; i < p.children.length; i++) {
			if (p.children[i] === el) {
				return ___path('', p) + '/' + i;
			}
		}
	} else {
	}
};

// https://developer.mozilla.org/en-US/docs/Web/API/TextEncoder
// CC0
if (typeof TextEncoder === "undefined") {
    TextEncoder=function TextEncoder(){};
    TextEncoder.prototype.encode = function encode(str) {
        "use strict";
        var Len = str.length, resPos = -1;
        // The Uint8Array's length must be at least 3x the length of the string because an invalid UTF-16
        //  takes up the equivelent space of 3 UTF-8 characters to encode it properly. However, Array's
        //  have an auto expanding length and 1.5x should be just the right balance for most uses.
        var resArr = typeof Uint8Array === "undefined" ? new Array(Len * 1.5) : new Uint8Array(Len * 3);
        for (var point=0, nextcode=0, i = 0; i !== Len; ) {
            point = str.charCodeAt(i), i += 1;
            if (point >= 0xD800 && point <= 0xDBFF) {
                if (i === Len) {
                    resArr[resPos += 1] = 0xef/*0b11101111*/; resArr[resPos += 1] = 0xbf/*0b10111111*/;
                    resArr[resPos += 1] = 0xbd/*0b10111101*/; break;
                }
                // https://mathiasbynens.be/notes/javascript-encoding#surrogate-formulae
                nextcode = str.charCodeAt(i);
                if (nextcode >= 0xDC00 && nextcode <= 0xDFFF) {
                    point = (point - 0xD800) * 0x400 + nextcode - 0xDC00 + 0x10000;
                    i += 1;
                    if (point > 0xffff) {
                        resArr[resPos += 1] = (0x1e/*0b11110*/<<3) | (point>>>18);
                        resArr[resPos += 1] = (0x2/*0b10*/<<6) | ((point>>>12)&0x3f/*0b00111111*/);
                        resArr[resPos += 1] = (0x2/*0b10*/<<6) | ((point>>>6)&0x3f/*0b00111111*/);
                        resArr[resPos += 1] = (0x2/*0b10*/<<6) | (point&0x3f/*0b00111111*/);
                        continue;
                    }
                } else {
                    resArr[resPos += 1] = 0xef/*0b11101111*/; resArr[resPos += 1] = 0xbf/*0b10111111*/;
                    resArr[resPos += 1] = 0xbd/*0b10111101*/; continue;
                }
            }
            if (point <= 0x007f) {
                resArr[resPos += 1] = (0x0/*0b0*/<<7) | point;
            } else if (point <= 0x07ff) {
                resArr[resPos += 1] = (0x6/*0b110*/<<5) | (point>>>6);
                resArr[resPos += 1] = (0x2/*0b10*/<<6)  | (point&0x3f/*0b00111111*/);
            } else {
                resArr[resPos += 1] = (0xe/*0b1110*/<<4) | (point>>>12);
                resArr[resPos += 1] = (0x2/*0b10*/<<6)    | ((point>>>6)&0x3f/*0b00111111*/);
                resArr[resPos += 1] = (0x2/*0b10*/<<6)    | (point&0x3f/*0b00111111*/);
            }
        }
        if (typeof Uint8Array !== "undefined") return resArr.subarray(0, resPos + 1);
        // else // IE 6-9
        resArr.length = resPos + 1; // trim off extra weight
        return resArr;
    };
    TextEncoder.prototype.toString = function(){return "[object TextEncoder]"};
    try { // Object.defineProperty only works on DOM prototypes in IE8
        Object.defineProperty(TextEncoder.prototype,"encoding",{
            get:function(){if(TextEncoder.prototype.isPrototypeOf(this)) return"utf-8";
                           else throw TypeError("Illegal invocation");}
        });
    } catch(e) { /*IE6-8 fallback*/ TextEncoder.prototype.encoding = "utf-8"; }
    if(typeof Symbol!=="undefined")TextEncoder.prototype[Symbol.toStringTag]="TextEncoder";
}

function LocalStorage() {
        var data = {};
        this.setItem = function(id, val) {
                return data[id] = String(val);
        };
        this.getItem = function(id) {
                return data.hasOwnProperty(id) ? data[id] : undefined;
        };
        this.removeItem = function(id) {
                return delete data[id];
        };
        this.clear = function() {
                return data = {};
        };
}
window.localStorage = new LocalStorage();
window.sessionStorage = new LocalStorage();
