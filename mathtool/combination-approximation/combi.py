#!/usr/bin/env python
# coding: utf-8

"""
Copyright (c) 2017 Temple3x (temple3x@gmail.com)

Use of this source code is governed by the MIT License
that can be found in the LICENSE file.
"""

import math

def facto(n):
    if n == 1:
        return 1
    return n*facto(n-1)

def combi(n, m):
    return facto(n) / facto(m) / facto(n-m)

def approx_combi(n, m):
    u = n * .5
    sig = (n*.5*.5) ** 0.5

    power = - (m-u) ** 2 / 2 / sig / sig
    return 1.0 / (sig * ((2*math.pi)**.5)) * (math.e ** power) * (2**n)
