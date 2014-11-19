package prelude

const goroutines = `
var $stackDepthOffset = 0;
var $getStackDepth = function() {
  var err = new Error();
  if (err.stack === undefined) {
    return undefined;
  }
  return $stackDepthOffset + err.stack.split("\n").length;
};

var $deferFrames = [], $skippedDeferFrames = 0, $jumpToDefer = false, $panicStackDepth = null, $panicValue;
var $callDeferred = function(deferred, jsErr) {
  if ($skippedDeferFrames !== 0) {
    $skippedDeferFrames--;
    throw jsErr;
  }
  if ($jumpToDefer) {
    $jumpToDefer = false;
    throw jsErr;
  }
  if (jsErr) {
    var newErr = null;
    try {
      $deferFrames.push(deferred);
      $panic(new $packages["github.com/gopherjs/gopherjs/js"].Error.Ptr(jsErr));
    } catch (err) {
      newErr = err;
    }
    $deferFrames.pop();
    $callDeferred(deferred, newErr);
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

  var call, localSkippedDeferFrames = 0;
  try {
    while (true) {
      if (deferred === null) {
        deferred = $deferFrames[$deferFrames.length - 1 - localSkippedDeferFrames];
        if (deferred === undefined) {
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
          var e = new Error(msg);
          if (localPanicValue.Stack !== undefined) {
            e.stack = localPanicValue.Stack();
            e.stack = msg + e.stack.substr(e.stack.indexOf("\n"));
          }
          throw e;
        }
      }
      var call = deferred.pop();
      if (call === undefined) {
        if (localPanicValue !== undefined) {
          localSkippedDeferFrames++;
          deferred = null;
          continue;
        }
        return;
      }
      var r = call[0].apply(undefined, call[1]);
      if (r && r.$blocking) {
        deferred.push([r, []]);
      }

      if (localPanicValue !== undefined && $panicStackDepth === null) {
        throw null; /* error was recovered */
      }
    }
  } finally {
    $skippedDeferFrames += localSkippedDeferFrames;
    if ($curGoroutine.asleep) {
      deferred.push(call);
      $jumpToDefer = true;
    }
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
  $callDeferred(null, null);
};
var $recover = function() {
  if ($panicStackDepth === null || ($panicStackDepth !== undefined && $panicStackDepth !== $getStackDepth() - 2)) {
    return $ifaceNil;
  }
  $panicStackDepth = null;
  return $panicValue;
};
var $nonblockingCall = function() {
  $panic(new $packages["runtime"].NotSupportedError.Ptr("non-blocking call to blocking function (mark call with \"//gopherjs:blocking\" to fix)"));
};
var $throw = function(err) { throw err; };
var $throwRuntimeError; /* set by package "runtime" */

var $dummyGoroutine = { asleep: false, exit: false, panicStack: [] };
var $curGoroutine = $dummyGoroutine, $totalGoroutines = 0, $awakeGoroutines = 0, $checkForDeadlock = true;
var $go = function(fun, args, direct) {
  $totalGoroutines++;
  $awakeGoroutines++;
  args.push(true);
  var goroutine = function() {
    var rescheduled = false;
    try {
      $curGoroutine = goroutine;
      $skippedDeferFrames = 0;
      $jumpToDefer = false;
      var r = fun.apply(undefined, args);
      if (r && r.$blocking) {
        fun = r;
        args = [];
        $schedule(goroutine, direct);
        rescheduled = true;
        return;
      }
      goroutine.exit = true;
    } catch (err) {
      if (!$curGoroutine.asleep) {
        goroutine.exit = true;
        throw err;
      }
    } finally {
      $curGoroutine = $dummyGoroutine;
      if (goroutine.exit && !rescheduled) { /* also set by runtime.Goexit() */
        $totalGoroutines--;
        goroutine.asleep = true;
      }
      if (goroutine.asleep && !rescheduled) {
        $awakeGoroutines--;
        if ($awakeGoroutines === 0 && $totalGoroutines !== 0 && $checkForDeadlock) {
          console.error("fatal error: all goroutines are asleep - deadlock!");
        }
      }
    }
  };
  goroutine.asleep = false;
  goroutine.exit = false;
  goroutine.panicStack = [];
  $schedule(goroutine, direct);
};

var $scheduled = [], $schedulerLoopActive = false;
var $schedule = function(goroutine, direct) {
  if (goroutine.asleep) {
    goroutine.asleep = false;
    $awakeGoroutines++;
  }

  if (direct) {
    goroutine();
    return;
  }

  $scheduled.push(goroutine);
  if (!$schedulerLoopActive) {
    $schedulerLoopActive = true;
    setTimeout(function() {
      while (true) {
        var r = $scheduled.shift();
        if (r === undefined) {
          $schedulerLoopActive = false;
          break;
        }
        r();
      };
    }, 0);
  }
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
  chan.$sendQueue.push(function() {
    $schedule(thisGoroutine);
    return value;
  });
  var blocked = false;
  var f = function() {
    if (blocked) {
      if (chan.$closed) {
        $throwRuntimeError("send on closed channel");
      }
      return;
    };
    blocked = true;
    $curGoroutine.asleep = true;
    throw null;
  };
  f.$blocking = true;
  return f;
};
var $recv = function(chan) {
  var queuedSend = chan.$sendQueue.shift();
  if (queuedSend !== undefined) {
    chan.$buffer.push(queuedSend());
  }
  var bufferedValue = chan.$buffer.shift();
  if (bufferedValue !== undefined) {
    return [bufferedValue, true];
  }
  if (chan.$closed) {
    return [chan.constructor.elem.zero(), false];
  }

  var thisGoroutine = $curGoroutine, value;
  var queueEntry = function(v) {
    value = v;
    $schedule(thisGoroutine);
  };
  chan.$recvQueue.push(queueEntry);
  var blocked = false;
  var f = function() {
    if (blocked) {
      return value;
    };
    blocked = true;
    $curGoroutine.asleep = true;
    throw null;
  };
  f.$blocking = true;
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
    queuedSend(); /* will panic because of closed channel */
  }
  while (true) {
    var queuedRecv = chan.$recvQueue.shift();
    if (queuedRecv === undefined) {
      break;
    }
    queuedRecv([chan.constructor.elem.zero(), false]);
  }
};
var $select = function(comms) {
  var ready = [], i;
  var selection = -1;
  for (i = 0; i < comms.length; i++) {
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
  var removeFromQueues = function() {
    for (i = 0; i < entries.length; i++) {
      var entry = entries[i];
      var queue = entry[0];
      var index = queue.indexOf(entry[1]);
      if (index !== -1) {
        queue.splice(index, 1);
      }
    }
  };
  for (i = 0; i < comms.length; i++) {
    (function(i) {
      var comm = comms[i];
      switch (comm.length) {
      case 1: /* recv */
        var queueEntry = function(value) {
          selection = [i, value];
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
          selection = [i];
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
  var blocked = false;
  var f = function() {
    if (blocked) {
      return selection;
    };
    blocked = true;
    $curGoroutine.asleep = true;
    throw null;
  };
  f.$blocking = true;
  return f;
};
`
