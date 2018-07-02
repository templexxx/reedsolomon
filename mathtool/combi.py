#!/usr/bin/env python
# coding: utf-8

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

for n, m in ((5, 2), (10, 3), (15, 6)):
    print combi(n, m), approx_combi(n, m)
