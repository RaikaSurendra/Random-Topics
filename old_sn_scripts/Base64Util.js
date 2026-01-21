var Base64Util = Class.create();
(function () {

    var CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    var BYTEMAP = CHARS.split('').map(function (x) {
        return x.charCodeAt(0);
    });

    function fromCodePoint(c) {
        return c < 0x10000 ? String.fromCharCode(c) : String.fromCharCode((c >> 10) + 0xD7C0) + String.fromCharCode((c & 0x3FF) + 0xDC00);
    }

    function bytesToString(b, pos, len) {
        if (pos < 0 || len < 0 || pos + len > b.length) {
            throw "Out of bounds";
        }
        var s = "";
        var i = pos;
        var max = pos + len;
        var encoding = 0;
        switch (encoding) {
            case 0: // UTF-8
                while (i < max) {
                    var c = b[i++];
                    if (c < 128) {
                        if (c == 0) {
                            break;
                        }
                        s += fromCodePoint(c);
                    } else if (c < 224) {
                        var code = (c & 63) << 6 | b[i++] & 127;
                        s += fromCodePoint(code);
                    } else if (c < 240) {
                        var code = (c & 31) << 12 | (b[i++] & 127) << 6 | b[i++] & 127;
                        s += fromCodePoint(code);
                    } else {
                        var u = (c & 15) << 18 | (b[i++] & 127) << 12 | (b[i++] & 127) << 6 | b[i++] & 127;
                        s += fromCodePoint(u);
                    }
                }
                break;
            case 1: // Raw Native
                while (i < max) {
                    var c = b[i++] | b[i++] << 8;
                    s += fromCodePoint(c);
                }
                break;
        }
        return s;
    }

    function BaseCode(byteMap) {
        var len = byteMap.length;
        var nbits = 1;
        while (len > 1 << nbits) ++nbits;
        if (nbits > 8 || len != 1 << nbits) {
            throw "BaseCode : base length must be a power of two.";
        }
        this.base = byteMap;
        this.nbits = nbits;

        this.encodeBytes = function (b) {
            var nbits = this.nbits;
            var base = this.base;
            var size = b.length * 8 / nbits | 0;
            //gs.addInfoMessage("Calculated size: "+b.length+"?: "+size);
            var out = new Array(size + (b.length * 8 % nbits == 0 ? 0 : 1));
            var buf = 0;
            var curbits = 0;
            var mask = (1 << nbits) - 1;
            var pin = 0;
            var pout = 0;
            while (pout < size) {
                while (curbits < nbits) {
                    curbits += 8;
                    buf <<= 8;
                    buf |= b[pin++];
                }
                curbits -= nbits;
                out[pout++] = base[buf >> curbits & mask];
            }
            if (curbits > 0) {
                out[pout++] = base[buf << nbits - curbits & mask];
            }
            return out;
        };
    }

    Base64Util.encode = function (bytes, complement) {
        if (complement == null) {
            complement = true;
        }
        var encodedBytes = new BaseCode(BYTEMAP).encodeBytes(bytes);
        var str = bytesToString(encodedBytes, 0, encodedBytes.length);
        if (complement) {
            switch (bytes.length % 3) {
                case 1:
                    str += "==";
                    break;
                case 2:
                    str += "=";
                    break;
                default:
            }
        }
        return str;
    };
})();