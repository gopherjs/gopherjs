package prelude

const goroutinesMinified = `var $panicValue,$stackDepthOffset=0,$getStackDepth=function(){var e=new Error;if(void 0!==e.stack)return $stackDepthOffset+e.stack.split("\n").length},$panicStackDepth=null,$callDeferred=function(e,r,n){if(!n&&null!==e&&e.index>=$curGoroutine.deferStack.length)throw r;if(null!==r){var t=null;try{$curGoroutine.deferStack.push(e),$panic(new $jsErrorPtr(r))}catch(e){t=e}return $curGoroutine.deferStack.pop(),void $callDeferred(e,t)}if(!$curGoroutine.asleep){$stackDepthOffset--;var o=$panicStackDepth,c=$panicValue,u=$curGoroutine.panicStack.pop();void 0!==u&&($panicStackDepth=$getStackDepth(),$panicValue=u);try{for(;;){if(null===e&&void 0===(e=$curGoroutine.deferStack[$curGoroutine.deferStack.length-1])){if($panicStackDepth=null,u.Object instanceof Error)throw u.Object;var i;throw i=u.constructor===$String?u.$val:void 0!==u.Error?u.Error():void 0!==u.String?u.String():u,new Error(i)}var a=e.pop();if(void 0===a){if($curGoroutine.deferStack.pop(),void 0!==u){e=null;continue}return}var l=a[0].apply(a[2],a[1]);if(l&&void 0!==l.$blk){if(e.push([l.$blk,[],l]),n)throw null;return}if(void 0!==u&&null===$panicStackDepth)throw null}}finally{void 0!==u&&(null!==$panicStackDepth&&$curGoroutine.panicStack.push(u),$panicStackDepth=o,$panicValue=c),$stackDepthOffset++}}},$panic=function(e){$curGoroutine.panicStack.push(e),$callDeferred(null,null,!0)},$recover=function(){return null===$panicStackDepth||void 0!==$panicStackDepth&&$panicStackDepth!==$getStackDepth()-2?$ifaceNil:($panicStackDepth=null,$panicValue)},$throw=function(e){throw e},$noGoroutine={asleep:!1,exit:!1,deferStack:[],panicStack:[]},$curGoroutine=$noGoroutine,$totalGoroutines=0,$awakeGoroutines=0,$checkForDeadlock=!0,$mainFinished=!1,$go=function(e,r){$totalGoroutines++,$awakeGoroutines++;var n=function(){try{$curGoroutine=n;var t=e.apply(void 0,r);if(t&&void 0!==t.$blk)return e=function(){return t.$blk()},void(r=[]);n.exit=!0}catch(e){if(!n.exit)throw e}finally{$curGoroutine=$noGoroutine,n.exit&&($totalGoroutines--,n.asleep=!0),n.asleep&&($awakeGoroutines--,!$mainFinished&&0===$awakeGoroutines&&$checkForDeadlock&&(console.error("fatal error: all goroutines are asleep - deadlock!"),void 0!==$global.process&&$global.process.exit(2)))}};n.asleep=!1,n.exit=!1,n.deferStack=[],n.panicStack=[],$schedule(n)},$scheduled=[],$runScheduled=function(){try{for(var e;void 0!==(e=$scheduled.shift());)e()}finally{$scheduled.length>0&&setTimeout($runScheduled,0)}},$schedule=function(e){e.asleep&&(e.asleep=!1,$awakeGoroutines++),$scheduled.push(e),$curGoroutine===$noGoroutine&&$runScheduled()},$setTimeout=function(e,r){return $awakeGoroutines++,setTimeout(function(){$awakeGoroutines--,e()},r)},$block=function(){$curGoroutine===$noGoroutine&&$throwRuntimeError("cannot block in JavaScript callback, fix by wrapping code in goroutine"),$curGoroutine.asleep=!0},$send=function(e,r){e.$closed&&$throwRuntimeError("send on closed channel");var n=e.$recvQueue.shift();if(void 0===n){if(!(e.$buffer.length<e.$capacity)){var t,o=$curGoroutine;return e.$sendQueue.push(function(e){return t=e,$schedule(o),r}),$block(),{$blk:function(){t&&$throwRuntimeError("send on closed channel")}}}e.$buffer.push(r)}else n([r,!0])},$recv=function(e){var r=e.$sendQueue.shift();void 0!==r&&e.$buffer.push(r(!1));var n=e.$buffer.shift();if(void 0!==n)return[n,!0];if(e.$closed)return[e.$elem.zero(),!1];var t=$curGoroutine,o={$blk:function(){return this.value}};return e.$recvQueue.push(function(e){o.value=e,$schedule(t)}),$block(),o},$close=function(e){for(e.$closed&&$throwRuntimeError("close of closed channel"),e.$closed=!0;;){var r=e.$sendQueue.shift();if(void 0===r)break;r(!0)}for(;;){var n=e.$recvQueue.shift();if(void 0===n)break;n([e.$elem.zero(),!1])}},$select=function(e){for(var r=[],n=-1,t=0;t<e.length;t++){var o,c=(o=e[t])[0];switch(o.length){case 0:n=t;break;case 1:(0!==c.$sendQueue.length||0!==c.$buffer.length||c.$closed)&&r.push(t);break;case 2:c.$closed&&$throwRuntimeError("send on closed channel"),(0!==c.$recvQueue.length||c.$buffer.length<c.$capacity)&&r.push(t)}}if(0!==r.length&&(n=r[Math.floor(Math.random()*r.length)]),-1!==n)switch((o=e[n]).length){case 0:return[n];case 1:return[n,$recv(o[0])];case 2:return $send(o[0],o[1]),[n]}var u=[],i=$curGoroutine,a={$blk:function(){return this.selection}},l=function(){for(var e=0;e<u.length;e++){var r=u[e],n=r[0],t=n.indexOf(r[1]);-1!==t&&n.splice(t,1)}};for(t=0;t<e.length;t++)!function(r){var n=e[r];switch(n.length){case 1:var t=function(e){a.selection=[r,e],l(),$schedule(i)};u.push([n[0].$recvQueue,t]),n[0].$recvQueue.push(t);break;case 2:t=function(){return n[0].$closed&&$throwRuntimeError("send on closed channel"),a.selection=[r],l(),$schedule(i),n[1]};u.push([n[0].$sendQueue,t]),n[0].$sendQueue.push(t)}}(t);return $block(),a};`
const goroutines = `
var $stackDepthOffset = 0;
var $getStackDepth = function() {
  var err = new Error();
  if (err.stack === undefined) {
    return undefined;
  }
  return $stackDepthOffset + err.stack.split("\n").length;
};

var $panicStackDepth = null, $panicValue;
var $callDeferred = function(deferred, jsErr, fromPanic) {
  if (!fromPanic && deferred !== null && deferred.index >= $curGoroutine.deferStack.length) {
    throw jsErr;
  }
  if (jsErr !== null) {
    var newErr = null;
    try {
      $curGoroutine.deferStack.push(deferred);
      $panic(new $jsErrorPtr(jsErr));
    } catch (err) {
      newErr = err;
    }
    $curGoroutine.deferStack.pop();
    $callDeferred(deferred, newErr);
    return;
  }
  if ($curGoroutine.asleep) {
    return;
  }

  $stackDepthOffset--;
  var outerPanicStackDepth = $panicStackDepth;
  var outerPanicValue = $panicValue;

  var localPanicValue = $curGoroutine.panicStack.pop();
  if (localPanicValue !== undefined) {
    $panicStackDepth = $getStackDepth();
    $panicValue = localPanicValue;
  }

  try {
    while (true) {
      if (deferred === null) {
        deferred = $curGoroutine.deferStack[$curGoroutine.deferStack.length - 1];
        if (deferred === undefined) {
          /* The panic reached the top of the stack. Clear it and throw it as a JavaScript error. */
          $panicStackDepth = null;
          if (localPanicValue.Object instanceof Error) {
            throw localPanicValue.Object;
          }
          var msg;
          if (localPanicValue.constructor === $String) {
            msg = localPanicValue.$val;
          } else if (localPanicValue.Error !== undefined) {
            msg = localPanicValue.Error();
          } else if (localPanicValue.String !== undefined) {
            msg = localPanicValue.String();
          } else {
            msg = localPanicValue;
          }
          throw new Error(msg);
        }
      }
      var call = deferred.pop();
      if (call === undefined) {
        $curGoroutine.deferStack.pop();
        if (localPanicValue !== undefined) {
          deferred = null;
          continue;
        }
        return;
      }
      var r = call[0].apply(call[2], call[1]);
      if (r && r.$blk !== undefined) {
        deferred.push([r.$blk, [], r]);
        if (fromPanic) {
          throw null;
        }
        return;
      }

      if (localPanicValue !== undefined && $panicStackDepth === null) {
        throw null; /* error was recovered */
      }
    }
  } finally {
    if (localPanicValue !== undefined) {
      if ($panicStackDepth !== null) {
        $curGoroutine.panicStack.push(localPanicValue);
      }
      $panicStackDepth = outerPanicStackDepth;
      $panicValue = outerPanicValue;
    }
    $stackDepthOffset++;
  }
};

var $panic = function(value) {
  $curGoroutine.panicStack.push(value);
  $callDeferred(null, null, true);
};
var $recover = function() {
  if ($panicStackDepth === null || ($panicStackDepth !== undefined && $panicStackDepth !== $getStackDepth() - 2)) {
    return $ifaceNil;
  }
  $panicStackDepth = null;
  return $panicValue;
};
var $throw = function(err) { throw err; };

var $noGoroutine = { asleep: false, exit: false, deferStack: [], panicStack: [] };
var $curGoroutine = $noGoroutine, $totalGoroutines = 0, $awakeGoroutines = 0, $checkForDeadlock = true;
var $mainFinished = false;
var $go = function(fun, args) {
  $totalGoroutines++;
  $awakeGoroutines++;
  var $goroutine = function() {
    try {
      $curGoroutine = $goroutine;
      var r = fun.apply(undefined, args);
      if (r && r.$blk !== undefined) {
        fun = function() { return r.$blk(); };
        args = [];
        return;
      }
      $goroutine.exit = true;
    } catch (err) {
      if (!$goroutine.exit) {
        throw err;
      }
    } finally {
      $curGoroutine = $noGoroutine;
      if ($goroutine.exit) { /* also set by runtime.Goexit() */
        $totalGoroutines--;
        $goroutine.asleep = true;
      }
      if ($goroutine.asleep) {
        $awakeGoroutines--;
        if (!$mainFinished && $awakeGoroutines === 0 && $checkForDeadlock) {
          console.error("fatal error: all goroutines are asleep - deadlock!");
          if ($global.process !== undefined) {
            $global.process.exit(2);
          }
        }
      }
    }
  };
  $goroutine.asleep = false;
  $goroutine.exit = false;
  $goroutine.deferStack = [];
  $goroutine.panicStack = [];
  $schedule($goroutine);
};

var $scheduled = [];
var $runScheduled = function() {
  try {
    var r;
    while ((r = $scheduled.shift()) !== undefined) {
      r();
    }
  } finally {
    if ($scheduled.length > 0) {
      setTimeout($runScheduled, 0);
    }
  }
};

var $schedule = function(goroutine) {
  if (goroutine.asleep) {
    goroutine.asleep = false;
    $awakeGoroutines++;
  }
  $scheduled.push(goroutine);
  if ($curGoroutine === $noGoroutine) {
    $runScheduled();
  }
};

var $setTimeout = function(f, t) {
  $awakeGoroutines++;
  return setTimeout(function() {
    $awakeGoroutines--;
    f();
  }, t);
};

var $block = function() {
  if ($curGoroutine === $noGoroutine) {
    $throwRuntimeError("cannot block in JavaScript callback, fix by wrapping code in goroutine");
  }
  $curGoroutine.asleep = true;
};

var $send = function(chan, value) {
  if (chan.$closed) {
    $throwRuntimeError("send on closed channel");
  }
  var queuedRecv = chan.$recvQueue.shift();
  if (queuedRecv !== undefined) {
    queuedRecv([value, true]);
    return;
  }
  if (chan.$buffer.length < chan.$capacity) {
    chan.$buffer.push(value);
    return;
  }

  var thisGoroutine = $curGoroutine;
  var closedDuringSend;
  chan.$sendQueue.push(function(closed) {
    closedDuringSend = closed;
    $schedule(thisGoroutine);
    return value;
  });
  $block();
  return {
    $blk: function() {
      if (closedDuringSend) {
        $throwRuntimeError("send on closed channel");
      }
    }
  };
};
var $recv = function(chan) {
  var queuedSend = chan.$sendQueue.shift();
  if (queuedSend !== undefined) {
    chan.$buffer.push(queuedSend(false));
  }
  var bufferedValue = chan.$buffer.shift();
  if (bufferedValue !== undefined) {
    return [bufferedValue, true];
  }
  if (chan.$closed) {
    return [chan.$elem.zero(), false];
  }

  var thisGoroutine = $curGoroutine;
  var f = { $blk: function() { return this.value; } };
  var queueEntry = function(v) {
    f.value = v;
    $schedule(thisGoroutine);
  };
  chan.$recvQueue.push(queueEntry);
  $block();
  return f;
};
var $close = function(chan) {
  if (chan.$closed) {
    $throwRuntimeError("close of closed channel");
  }
  chan.$closed = true;
  while (true) {
    var queuedSend = chan.$sendQueue.shift();
    if (queuedSend === undefined) {
      break;
    }
    queuedSend(true); /* will panic */
  }
  while (true) {
    var queuedRecv = chan.$recvQueue.shift();
    if (queuedRecv === undefined) {
      break;
    }
    queuedRecv([chan.$elem.zero(), false]);
  }
};
var $select = function(comms) {
  var ready = [];
  var selection = -1;
  for (var i = 0; i < comms.length; i++) {
    var comm = comms[i];
    var chan = comm[0];
    switch (comm.length) {
    case 0: /* default */
      selection = i;
      break;
    case 1: /* recv */
      if (chan.$sendQueue.length !== 0 || chan.$buffer.length !== 0 || chan.$closed) {
        ready.push(i);
      }
      break;
    case 2: /* send */
      if (chan.$closed) {
        $throwRuntimeError("send on closed channel");
      }
      if (chan.$recvQueue.length !== 0 || chan.$buffer.length < chan.$capacity) {
        ready.push(i);
      }
      break;
    }
  }

  if (ready.length !== 0) {
    selection = ready[Math.floor(Math.random() * ready.length)];
  }
  if (selection !== -1) {
    var comm = comms[selection];
    switch (comm.length) {
    case 0: /* default */
      return [selection];
    case 1: /* recv */
      return [selection, $recv(comm[0])];
    case 2: /* send */
      $send(comm[0], comm[1]);
      return [selection];
    }
  }

  var entries = [];
  var thisGoroutine = $curGoroutine;
  var f = { $blk: function() { return this.selection; } };
  var removeFromQueues = function() {
    for (var i = 0; i < entries.length; i++) {
      var entry = entries[i];
      var queue = entry[0];
      var index = queue.indexOf(entry[1]);
      if (index !== -1) {
        queue.splice(index, 1);
      }
    }
  };
  for (var i = 0; i < comms.length; i++) {
    (function(i) {
      var comm = comms[i];
      switch (comm.length) {
      case 1: /* recv */
        var queueEntry = function(value) {
          f.selection = [i, value];
          removeFromQueues();
          $schedule(thisGoroutine);
        };
        entries.push([comm[0].$recvQueue, queueEntry]);
        comm[0].$recvQueue.push(queueEntry);
        break;
      case 2: /* send */
        var queueEntry = function() {
          if (comm[0].$closed) {
            $throwRuntimeError("send on closed channel");
          }
          f.selection = [i];
          removeFromQueues();
          $schedule(thisGoroutine);
          return comm[1];
        };
        entries.push([comm[0].$sendQueue, queueEntry]);
        comm[0].$sendQueue.push(queueEntry);
        break;
      }
    })(i);
  }
  $block();
  return f;
};
`
