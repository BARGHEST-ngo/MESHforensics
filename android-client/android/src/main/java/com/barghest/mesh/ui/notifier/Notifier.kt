// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause
//
// Portions Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file contains code originally from Tailscale (BSD-3-Clause)
// with modifications by BARGHEST. The modified version is licensed
// under AGPL-3.0-or-later. See LICENSE for details.

package com.barghest.mesh.ui.notifier

import com.barghest.mesh.ui.model.Health
import com.barghest.mesh.ui.model.Ipn
import com.barghest.mesh.ui.model.Netmap
import com.barghest.mesh.util.TSLog
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.serialization.ExperimentalSerializationApi

// Notifier is a wrapper around the IPN Bus notifier.  It provides a way to watch
// for changes in various parts of the Tailscale engine.  You will typically only use
// a single Notifier per instance of your application which lasts for the lifetime of
// the process.
//
// The primary entry point here is watchIPNBus which will start a watcher on the IPN bus
// and return you the session Id.  When you are done with your watcher, you must call
// unwatchIPNBus with the sessionId.
object Notifier {
  private val TAG = Notifier::class.simpleName
  private val _state = MutableStateFlow(Ipn.State.NeedsLogin)
  val state: StateFlow<Ipn.State> = _state
  val netmap: StateFlow<Netmap.NetworkMap?> = MutableStateFlow(null)
  val prefs: StateFlow<Ipn.Prefs?> = MutableStateFlow(null)
  val version: StateFlow<String?> = MutableStateFlow(null)
  val health: StateFlow<Health.State?> = MutableStateFlow(null)

  // Taildrop-specific State
  val outgoingFiles: StateFlow<List<Ipn.OutgoingFile>?> = MutableStateFlow(null)

  @Synchronized
  @OptIn(ExperimentalSerializationApi::class)
  fun start(scope: CoroutineScope) {
    TSLog.d(TAG, "Starting Notifier")
  }

  fun stop() {
    TSLog.d(TAG, "Stopping Notifier")
  }

  // NotifyWatchOpt is a bitmask of options supplied to the notifier to specify which
  // what we want to see on the Notify bus
  private enum class NotifyWatchOpt(val value: Int) {
    EngineUpdates(1),
    InitialState(2),
    Prefs(4),
    Netmap(8),
    NoPrivateKey(16),
    InitialTailFSShares(32),
    InitialOutgoingFiles(64),
    InitialHealthState(128),
    RateLimitNetmaps(256),
  }

  fun setState(newState: Ipn.State) {
    _state.value = newState
  }
}
