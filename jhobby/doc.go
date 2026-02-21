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
spline control point information. It may contain various parameters at
knots and/or joins. In the
MetaFont/MetaPost DSL one may specify it as follows:

   (0,0)..(2,3)..tension 1.4..(5,3)..(3,-1){left}..cycle

On evaluation of this path expression, MetaFont/MetaPost immediately will
find the control points in a clever way to construct a smooth curve through
the knots of the path. When using the methods of package "jhobby", clients will
build a skeleton path with a kind of builder pattern (package qualifiers
omitted for clarity and brevity):

   Nullpath().Knot(P(0,0)).Curve().Knot(P(2,3)).TensionCurve(N(1.4),N(1.4)).Knot(P(5,3))
      .Curve().DirKnot(P(3,-1),P(-1,0)).Curve().Cycle()

A built path is then subjected to a call to FindHobbyControls(...)

   controls, err := FindHobbyControls(path, nil)

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

Copyright (c) Norbert Pillmayer

All rights reserved.

Please refer to the license file for more information.
*/
package jhobby

import "fmt"

// AsString returns
// a path -- optionally including spline control points -- as a (debugging)
// string. The string contains newlines if control point information is present.
// Otherwise it will include the knot coordinates in one line.
//
// Example, a circle of diameter 1 around (2,1):
//
//	(1,1) .. controls (1.0000,1.5523) and (1.4477,2.0000)
//	  .. (2,2) .. controls (2.5523,2.0000) and (3.0000,1.5523)
//	  .. (3,1) .. controls (3.0000,0.4477) and (2.5523,0.0000)
//	  .. (2,0) .. controls (1.4477,0.0000) and (1.0000,0.4477)
//	  .. cycle
//
// The format is not fully equivalent to MetaFont's, but close.
func AsString(path *Path, contr *Controls) string {
	var s string
	for i := 0; i < path.N(); i++ {
		pt := path.Z(i)
		if i > 0 {
			if contr != nil {
				s += fmt.Sprintf(" and %s\n  .. ", ptstring(contr.PreControl(i), true))
			} else {
				s += " .. "
			}
		}
		s += fmt.Sprintf("%s", ptstring(pt, false))
		if contr != nil && (i < path.N()-1 || path.IsCycle()) {
			s += fmt.Sprintf(" .. controls %s", ptstring(contr.PostControl(i), true))
		}
	}
	if path.IsCycle() {
		if contr != nil {
			s += fmt.Sprintf(" and %s\n ", ptstring(contr.PreControl(0), true))
		}
		s += " .. cycle"
	}
	return s
}
