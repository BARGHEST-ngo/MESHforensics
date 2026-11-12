// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause
//
// Portions Copyright (c) BARGHEST
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This file contains code originally from Tailscale (BSD-3-Clause)
// with modifications by BARGHEST. The modified version is licensed
// under AGPL-3.0-or-later. See LICENSE for details.

package com.barghest.mesh

import android.net.IpPrefix as AndroidIpPrefix
import android.net.VpnService
import android.os.Build
import java.net.InetAddress

class VPNServiceBuilder(private val builder: VpnService.Builder) {
  fun addAddress(p0: String, p1: Int) {
    builder.addAddress(p0, p1)
  }

  fun addDNSServer(p0: String) {
    builder.addDnsServer(p0)
  }

  fun addRoute(p0: String, p1: Int) {
    builder.addRoute(p0, p1)
  }

  fun excludeRoute(p0: String, p1: Int) {
    // Only run this for API level 33 and up
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
      val inetAddress = InetAddress.getByName(p0)
      val prefix = AndroidIpPrefix(inetAddress, p1)
      builder.excludeRoute(prefix)
    }
  }

  fun setMTU(p0: Int) {
    builder.setMtu(p0)
  }

  fun establishVpn(): android.os.ParcelFileDescriptor? = builder.establish()
}
