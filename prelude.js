var Slice = function(array) {
  this.array = array
  this.offset = 0
  this.length = array.length
}

Slice.prototype = {
  get: function(index) {
    return this.array[this.offset + index];
  },
  set: function(index, value) {
    this.array[this.offset + index] = value;
  },
  subslice: function(begin, end) {
    var s = new Slice(this.array);
    s.offset = this.offset + begin;
    s.length = this.length - begin;
    if (end !== undefined) {
      s.length = end - begin;
    }
    return s;
  },
  toArray: function() {
    return this.array.slice(this.offset, this.offset + this.length);
  },
};

String.prototype.toSlice = function() {
  var array = new Uint8Array(this.length);
  for (var i = 0; i < this.length; i++) {
    array[i] = this.charCodeAt(i);
  }
  return new Slice(array)
};

var len = function(slice) {
  return slice.length;
}

var cap = function(slice) {
  return slice.array.length;
}

var make = function(constructor, length, capacity) {
  if (capacity === undefined) {
    capacity = length;
  }
  var slice = new Slice(new constructor(capacity));
  slice.length = length;
  return slice;
}

var append = function(slice, toAppend) {
  var newArray = slice.array;
  var newOffset = slice.offset;
  var newLength = slice.length + toAppend.length;

  if (slice.offset + newLength > newArray.length) {
    newArray = new newArray.constructor(newLength);
    for (var i = 0; i < slice.length; i++) {
      newArray[i] = slice.array[slice.offset + i];
    }
    newOffset = 0;
  }

  for (var i = 0; i < toAppend.length; i++) {
    newArray[newOffset + slice.length + i] = toAppend.get(i);
  }

  var newSlice = new Slice(newArray);
  newSlice.offset = newOffset;
  newSlice.length = newLength;
  return newSlice;
};

var string = function(s) {
  return String.fromCharCode(s);
};

var panic = function(msg) {
  throw msg;
};

var fmt = {
  Println: function(parts) {
    console.log.apply(console, parts.toArray());
  }
};

var reflect = {
  DeepEqual: function(a, b) {
    return JSON.stringify(a) === JSON.stringify(b) 
  }
};

