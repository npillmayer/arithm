// Package jhobby deals with MetaFont/MetaPost-like paths. It provides an
// implementation of John Hobby's spline interpolation algorithm.
/*

Spline interpolation by Hobby's algorithm results in aesthetically pleasing
curves superior to "normal" spline interpolation (as used in many graphics
programs). The primary source of information for "Hobby-splines" is:

   Smooth, Easy to Compute Interpolating Splines -- John D. Hobby
   Computer Science Dept. Stanford University
   Report No. STAN-CS-85-1047, Jan 1985
   http://i.stanford.edu/pub/cstr/reports/cs/tr/85/1047/CS-TR-85-1047.pdf

The practical algorithm is explained in

   Computers & Typesetting, Vol. B & D.
   http://www-cs-faculty.stanford.edu/~knuth/abcde.html

A good discussion of the implementation may be found in:

   (1) Implementing Hobby Curve
       posted on 2015-04-28 by Hui Zhou
       Perl code embedded at http://hz2.org/blog/hobby_curve.html
       (no copyright information)

Other implementations are available in Python:

   (2) Curve through a sequence of points with Metapost and TikZ
       https://tex.stackexchange.com/questions/54771/curve-through-a-sequence-of-points-with-metapost-and-tikz
       (Python code (c) Copyright 2012 JL Diaz)

and

   (3) Module metapost.path -- PyX Manual 0.14.1
       http://pyx.sourceforge.net/manual/metapost.html
       (GNU license (c) Copyright the PyX team)

This Go implementation is not the result of transcoding any of these
implementations, but it is of course inspired by them. The notation
sticks closely to the original code in MetaFont. The API's concept for
path building very loosely follows the ideas in PyX.

Usage

Clients of the package usually build a "skeleton" path, without any
spline control point information. Such a path is called a "HobbyPath" and
it may contain various parameters at knots and/or joins. In the
MetaFont/MetaPost DSL one may specify it as follows:

   (0,0)..(2,3)..tension 1.4..(5,3)..(3,-1){left}..cycle

On evaluation of this path expression, MetaFont/MetaPost immediately will
find the control points in a clever way to construct a smooth curve through
the knots of the path. When using the methods of package "path", clients will
build a skeleton path with a kind of builder pattern (package qualifiers
omitted for clarity and brevity):

   Nullpath().Knot(P(0,0)).Curve().Knot(P(2,3)).TensionCurve(N(1.4),N(1.4)).Knot(P(5,3))
      .Curve().DirKnot(P(3,-1),P(-1,0)).Curve().Cycle()

Alternatively clients may put interface HobbyCurve over their own path
data structure. Either way, a HobbyPath will then be subjected to a call to
FindHobbyControls(...)

   controls = FindHobbyControls(path, nil)

which returns the necessary control point information to produce a smooth
curve:

  (0,0) .. controls (-0.5882,1.2616) and (0.4229,2.6442)
   .. (2,3) .. controls (2.7160,3.1616) and (4.3325,3.2937)
   .. (5,3) .. controls (6.5505,2.3177) and (6.2401,-0.4348)
   .. (3,-1) .. controls (1.8036,-1.2085) and (0.4731,-1.0144)
   .. cycle

Caveats

(1) The development of this package is still in a very early phase.
Please do use with caution!

(2) Currently there are slight deviations from MetaFont's calculation,
probably due to different rounding. These are under investigation.


(3) Currently it isn't possible to explicitly set control points,
as I don't need this functionality. This may or may not change in the future.
Please note that the goal of this project is ultimately to support graphical
requirements for typesetting, not implementing a graphical system. If you
need a full fledged engine for preparing illustrations, you should stick
to MetaPost, which is a really great piece of software!


BSD License

Copyright (c) 2017–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. */
package jhobby
