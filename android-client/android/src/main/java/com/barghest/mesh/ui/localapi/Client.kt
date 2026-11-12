// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause
//
// Portions Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file contains code originally from Tailscale (BSD-3-Clause)
// with modifications by BARGHEST. The modified version is licensed
// under AGPL-3.0-or-later. See LICENSE for details.

package com.barghest.mesh.ui.localapi

import android.content.Context
import com.barghest.mesh.ui.model.BugReportID
import com.barghest.mesh.ui.model.Ipn
import com.barghest.mesh.ui.model.IpnLocal
import com.barghest.mesh.ui.model.IpnState
import com.barghest.mesh.ui.model.StableNodeID
import com.barghest.mesh.ui.model.Tailcfg
import com.barghest.mesh.util.TSLog
import kotlin.reflect.KType
import kotlin.reflect.typeOf
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private object Endpoint {
  const val BUG_REPORT = "bugreport"
  const val PREFS = "prefs"
  const val FILE_TARGETS = "file-targets"
  const val START = "start"
  const val LOGIN_INTERACTIVE = "login-interactive"
  const val RESET_AUTH = "reset-auth"
  const val LOGOUT = "logout"
  const val PROFILES = "profiles/"
  const val PROFILES_CURRENT = "profiles/current"
  const val STATUS = "status"
  const val TKA_STATUS = "tka/status"
  const val PING = "ping"
  const val ENABLE_EXIT_NODE = "set-use-exit-node-enabled"
}

typealias StatusResponseHandler = (Result<IpnState.Status>) -> Unit

typealias TailnetLockStatusResponseHandler = (Result<IpnState.NetworkLockStatus>) -> Unit

typealias BugReportIdHandler = (Result<BugReportID>) -> Unit

typealias PrefsHandler = (Result<Ipn.Prefs>) -> Unit

typealias PingResultHandler = (Result<IpnState.PingResult>) -> Unit

// FAKE(meshnet): the error every LocalAPI request fails with while there is no backend.
private fun noBackendError() = Exception("no backend: libtailscale removed, MESHnet not wired in yet")

/**
 * Client provides a mechanism for calling Go's LocalAPIClient. Every LocalAPI endpoint has a
 * corresponding method on this Client.
 */
class Client(private val scope: CoroutineScope) {
  private val TAG = Client::class.simpleName

  fun start(options: Ipn.Options, responseHandler: (Result<Unit>) -> Unit) {
    val body = Json.encodeToString(options).toByteArray()
    return post(Endpoint.START, body, responseHandler = responseHandler)
  }

  fun status(responseHandler: StatusResponseHandler) {
    get(Endpoint.STATUS, responseHandler = responseHandler)
  }

  fun ping(peer: Tailcfg.Node, responseHandler: PingResultHandler) {
    val ip = peer.primaryIPv4Address.orEmpty()
    if (ip.isEmpty()) {
      responseHandler(Result.failure(Exception("No IP address for peer $peer")))
      return
    }

    val path = "${Endpoint.PING}?ip=${ip}&type=disco"
    post(path, timeoutMillis = 2000L, responseHandler = responseHandler)
  }

  fun bugReportId(responseHandler: BugReportIdHandler) {
    post(Endpoint.BUG_REPORT, responseHandler = responseHandler)
  }

  fun prefs(responseHandler: PrefsHandler) {
    get(Endpoint.PREFS, responseHandler = responseHandler)
  }

  fun editPrefs(prefs: Ipn.MaskedPrefs, responseHandler: (Result<Ipn.Prefs>) -> Unit) {
    // Log the stack trace for debugging purposes for
    // https://github.com/tailscale/tailscale/issues/14125
    val stackTrace =
        Thread.currentThread().stackTrace.joinToString("\n") { element ->
          "at ${element.className}.${element.methodName}(${element.fileName}:${element.lineNumber})"
        }
    TSLog.v("editPrefs", "Called editPrefs with prefs: $prefs\nStack trace:\n$stackTrace")

    val body = Json.encodeToString(prefs).toByteArray()
    return patch(Endpoint.PREFS, body, responseHandler = responseHandler)
  }

  fun setUseExitNode(use: Boolean, responseHandler: (Result<Ipn.Prefs>) -> Unit) {
    val path = "${Endpoint.ENABLE_EXIT_NODE}?enabled=$use"
    return post(path, responseHandler = responseHandler)
  }

  fun profiles(responseHandler: (Result<List<IpnLocal.LoginProfile>>) -> Unit) {
    get(Endpoint.PROFILES, responseHandler = responseHandler)
  }

  fun currentProfile(responseHandler: (Result<IpnLocal.LoginProfile>) -> Unit) {
    return get(Endpoint.PROFILES_CURRENT, responseHandler = responseHandler)
  }

  fun addProfile(responseHandler: (Result<String>) -> Unit = {}) {
    return put(Endpoint.PROFILES, responseHandler = responseHandler)
  }

  fun deleteProfile(
      profile: IpnLocal.LoginProfile,
      responseHandler: (Result<String>) -> Unit = {}
  ) {
    return delete(Endpoint.PROFILES + profile.ID, responseHandler = responseHandler)
  }

  fun switchProfile(
      profile: IpnLocal.LoginProfile,
      responseHandler: (Result<String>) -> Unit = {}
  ) {
    return post(Endpoint.PROFILES + profile.ID, responseHandler = responseHandler)
  }

  fun startLoginInteractive(responseHandler: (Result<Unit>) -> Unit) {
    return post(Endpoint.LOGIN_INTERACTIVE, responseHandler = responseHandler)
  }

  fun logout(responseHandler: (Result<String>) -> Unit) {
    return post(Endpoint.LOGOUT, responseHandler = responseHandler)
  }

  fun tailnetLockStatus(responseHandler: TailnetLockStatusResponseHandler) {
    get(Endpoint.TKA_STATUS, responseHandler = responseHandler)
  }

  fun fileTargets(responseHandler: (Result<List<Ipn.FileTarget>>) -> Unit) {
    get(Endpoint.FILE_TARGETS, responseHandler = responseHandler)
  }

  fun resetAuth(responseHandler: (Result<Unit>) -> Unit) {
      return post(Endpoint.RESET_AUTH, responseHandler = responseHandler)
  }

  fun putTaildropFiles(
      context: Context,
      peerId: StableNodeID,
      files: Collection<Ipn.OutgoingFile>,
      responseHandler: (Result<String>) -> Unit
  ) {
    scope.launch { responseHandler(Result.failure(noBackendError())) }
  }

  private inline fun <reified T> get(
      path: String,
      body: ByteArray? = null,
      noinline responseHandler: (Result<T>) -> Unit
  ) {
    Request(
            scope = scope,
            method = "GET",
            path = path,
            body = body,
            responseType = typeOf<T>(),
            responseHandler = responseHandler)
        .execute()
  }

  private inline fun <reified T> put(
      path: String,
      body: ByteArray? = null,
      noinline responseHandler: (Result<T>) -> Unit
  ) {
    Request(
            scope = scope,
            method = "PUT",
            path = path,
            body = body,
            responseType = typeOf<T>(),
            responseHandler = responseHandler)
        .execute()
  }

  private inline fun <reified T> post(
      path: String,
      body: ByteArray? = null,
      timeoutMillis: Long = 30000,
      noinline responseHandler: (Result<T>) -> Unit
  ) {
    Request(
            scope = scope,
            method = "POST",
            path = path,
            body = body,
            timeoutMillis = timeoutMillis,
            responseType = typeOf<T>(),
            responseHandler = responseHandler)
        .execute()
  }

  private inline fun <reified T> patch(
      path: String,
      body: ByteArray? = null,
      noinline responseHandler: (Result<T>) -> Unit
  ) {
    Request(
            scope = scope,
            method = "PATCH",
            path = path,
            body = body,
            responseType = typeOf<T>(),
            responseHandler = responseHandler)
        .execute()
  }

  private inline fun <reified T> delete(
      path: String,
      noinline responseHandler: (Result<T>) -> Unit
  ) {
    Request(
            scope = scope,
            method = "DELETE",
            path = path,
            responseType = typeOf<T>(),
            responseHandler = responseHandler)
        .execute()
  }
}

class Request<T>(
    private val scope: CoroutineScope,
    private val method: String,
    path: String,
    private val body: ByteArray? = null,
    private val timeoutMillis: Long = 30000,
    private val responseType: KType,
    private val responseHandler: (Result<T>) -> Unit
) {
  private val fullPath = "/localapi/v0/$path"

  companion object {
    private const val TAG = "LocalAPIRequest"
    private val jsonDecoder = Json { ignoreUnknownKeys = true }
  }

  @OptIn(ExperimentalSerializationApi::class)
  fun execute() {
    // FAKE(meshnet): libtailscale is removed, so there is no LocalAPI to call.
    // Every request fails and callers take the failure path they already have for backend errors.
    TSLog.d(TAG, "FAKE(meshnet): no backend for request:${method}:${fullPath}")
    scope.launch { responseHandler(Result.failure(noBackendError())) }

  }
}
