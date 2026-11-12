// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause
//
// Portions Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file contains code originally from Tailscale (BSD-3-Clause)
// with modifications by BARGHEST. The modified version is licensed
// under AGPL-3.0-or-later. See LICENSE for details.
package com.barghest.mesh.util

import android.content.Context
import androidx.documentfile.provider.DocumentFile
import com.barghest.mesh.TaildropDirectoryStore
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow

object ShareFileHelper {
  private var appContext: Context? = null
  private var savedUri: String? = null
  private var scope: CoroutineScope? = null

  @JvmStatic
  fun init(context: Context, uri: String, appScope: CoroutineScope) {
    appContext = context.applicationContext
    savedUri = uri
    scope = appScope
    TSLog.d("ShareFileHelper", "init ShareFileHelper with savedUri: $savedUri")
  }

  val taildropPrompt = MutableSharedFlow<Unit>(replay = 1)

  fun observeTaildropPrompt(): Flow<Unit> = taildropPrompt

  @Volatile
  private var directoryReady: CompletableDeferred<Unit>? = null

  fun hasValidTaildropDir(): Boolean {
    val uri = TaildropDirectoryStore.loadSavedDir()
    if (uri == null) return false

    // Only SAF tree URIs are supported
    if (uri.scheme != "content") {
      TSLog.w("ShareFileHelper", "Invalid URI scheme for taildrop dir: ${uri.scheme}")
      return false
    }

    val context = appContext ?: return false
    val docFile = DocumentFile.fromTreeUri(context, uri)

    if (docFile == null || !docFile.exists() || !docFile.canWrite()) {
      TSLog.w("ShareFileHelper", "Stored taildrop URI is invalid or inaccessible: $uri")
      return false
    }

    return true
  }

  fun notifyDirectoryReady() {
    directoryReady?.takeIf { !it.isCompleted }?.complete(Unit)
  }

  fun setUri(uri: String) {
    savedUri = uri
  }
}
