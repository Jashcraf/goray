package goray

import (
	"math"
	"math/cmplx"
)

type Mat2C [2][2]complex128
type Vec2C [2]complex128

type NT struct {
	T float64    // thickness
	N complex128 // refactive index
}

type PolState int

const (
	Spol PolState = iota
	Ppol

	RadToDeg = 180 / math.Pi
	DegToRad = math.Pi / 180

	n1j = -complex(0, 1)
)

func MatMulVec2C(A Mat2C, B Vec2C) Vec2C {
	// TODO: not as fast as it could be, bug brandon about it later
	a := A[0][0]
	b := A[0][1]
	c := A[1][0]
	d := A[1][1]

	w := B[0]
	x := B[1]

	return Vec2C{
		a*w + b*x, c*w + d*x,
	}
}

func MatMul2C(A, B Mat2C) Mat2C {
	a := A[0][0]
	b := A[0][1]
	c := A[1][0]
	d := A[1][1]

	w := B[0][0]
	x := B[0][1]
	y := B[1][0]
	z := B[1][1]
	return Mat2C{
		{a*w + b*y, a*x + b*z},
		{c*w + d*y, c*x + d*z},
	}
}

func MatScale2C(A Mat2C, s complex128) Mat2C {
	return Mat2C{
		{A[0][0] * s, A[0][1] * s},
		{A[1][0] * s, A[1][1] * s},
	}
}

func CharacteristicMatrixS(lambda, d float64, n, theta complex128) Mat2C {
	k := complex(2*math.Pi/lambda, 0) * n
	dc := complex(d, 0)
	cost := cmplx.Cos(theta)
	beta := k * dc * cost
	sinb := cmplx.Sin(beta)
	cosb := cmplx.Cos(beta)
	upperRight := n1j * sinb / (cost * n)
	lowerLeft := n1j * n * cost * sinb
	return Mat2C{
		{cosb, upperRight},
		{lowerLeft, cosb},
	}
}

func CharacteristicMatrixP(lambda, d float64, n, theta complex128) Mat2C {
	k := complex(2*math.Pi/lambda, 0) * n
	dc := complex(d, 0)
	cost := cmplx.Cos(theta)
	beta := k * dc * cost
	sinb := cmplx.Sin(beta)
	cosb := cmplx.Cos(beta)
	upperRight := n1j * sinb * cost / n
	lowerLeft := n1j * n * sinb / cost
	return Mat2C{
		{cosb, upperRight},
		{lowerLeft, cosb},
	}
}

func Totalr(M Mat2C) complex128 {
	return M[1][0] / M[0][0]
}

func Totalt(M Mat2C) complex128 {
	return 1 / M[0][0]
}

func SnellAOR(n0, n1, theta complex128) complex128 {
	kernel := n0 / n1 * cmplx.Sin(theta)
	return cmplx.Asin(kernel)
}

func MultilayerStackrt(pol PolState, lambda float64, stack []NT, aoi float64, vacAmbient bool) (complex128, complex128) {
	var (
		n0    complex128
		n1    complex128
		theta complex128
		Amat  Mat2C
	)
	if len(stack) == 0 {
		panic("zero length stack is meaningless")
	}
	aoi = aoi * DegToRad

	if vacAmbient {
		n0 = 1
	} else {
		n0 = stack[0].N
	}
	theta = complex(aoi, 0)
	cos0 := math.Cos(aoi)
	cos0c := complex(cos0, 0)

	term1 := 1 / (2 * n0 * cos0c)
	n0cos0 := cos0c * n0
	front := Mat2C{
		{n0cos0, 1},
		{n0cos0, -1},
	}
	front = MatScale2C(front, term1)
	Amat = Mat2C{
		{1, 0},
		{0, 1},
	}
	if pol == Ppol {
		for _, nt := range stack {
			n1 = nt.N
			theta1 := SnellAOR(n0, n1, theta)
			Mj := CharacteristicMatrixP(lambda, nt.T, n1, theta1)
			Amat = MatMul2C(Amat, Mj)
			theta = theta1
			n0 = n1
		}
	} else if pol == Spol {
		for _, nt := range stack {
			n1 = nt.N
			theta1 := SnellAOR(n0, n1, theta)
			Mj := CharacteristicMatrixS(lambda, nt.T, n1, theta1)
			Amat = MatMul2C(Amat, Mj)
			theta = theta1
			n0 = n1
		}
	} else {
		panic("invalid polarization, must be either Ppol or Spol")
	}
	Amat = MatMul2C(front, Amat)
	// if vacAmbient {
	// 	n1 = complex(1, 0)
	// 	theta = SnellAOR(n0, n1, theta)
	// }
	cos1c := cmplx.Cos(theta)
	back := Mat2C{
		{1, 0},
		{n1 * cos1c, 0},
	}
	Amat = MatMul2C(Amat, back)
	return Totalr(Amat), Totalt(Amat)
}

func MacleodMatrixP(lambda, d float64, n, theta complex128) Mat2C {
	k := complex(2*math.Pi/lambda, 0) * n
	dc := complex(d, 0)
	cost := cmplx.Cos(theta)
	eta := n / cost
	delta := k * dc * cost
	cosd := cmplx.Cos(delta)
	upperRight := n1j * cmplx.Sin(delta) / eta
	lowerLeft := n1j * eta * cmplx.Sin(delta)
	return Mat2C{
		{cosd, upperRight},
		{lowerLeft, cosd},
	}

}

func MacleodMatrixS(lambda, d float64, n, theta complex128) Mat2C {
	k := complex(2*math.Pi/lambda, 0) * n
	dc := complex(d, 0)
	cost := cmplx.Cos(theta)
	eta := n * cost
	delta := k * dc * cost
	cosd := cmplx.Cos(delta)
	upperRight := n1j * cmplx.Sin(delta) / eta
	lowerLeft := n1j * eta * cmplx.Sin(delta)
	return Mat2C{
		{cosd, upperRight},
		{lowerLeft, cosd},
	}

}

func Macleodr(M Vec2C, eta0 complex128) complex128 {
	Y := M[1] / M[0]
	num := eta0 - Y
	den := eta0 + Y
	return num / den
}

func MacleodStackrt(pol PolState, lambda float64, stack []NT, aoi float64, vacAmbient bool) complex128 {
	// only returns r for now

	var (
		n0    complex128
		n1    complex128
		theta complex128
		eta0  complex128
		Amat  Mat2C
		Etam  Vec2C
	)

	if len(stack) == 0 {
		panic("zero length stack is meaningless")
	}
	aoi = aoi * DegToRad

	if vacAmbient {
		n0 = 1
	} else {
		n0 = stack[0].N
	}
	theta = complex(aoi, 0)
	cos0 := math.Cos(aoi)
	cos0c := complex(cos0, 0)
	Amat = Mat2C{
		{1, 0},
		{0, 1},
	}

	if pol == Ppol {
		// define eta ambient
		eta0 = n0 / cos0c

		for _, nt := range stack {
			n1 = nt.N
			theta1 := SnellAOR(n0, n1, theta)
			Mj := MacleodMatrixP(lambda, nt.T, n1, theta1)
			Amat = MatMul2C(Amat, Mj)
			theta = theta1
			n0 = n1
		}

		// define eta_medium
		eta := n1 / cos0c
		Etam = Vec2C{
			1, eta,
		}
	} else if pol == Spol {
		// define eta ambient
		eta0 = n0 * cos0c
		for _, nt := range stack {
			n1 = nt.N
			theta1 := SnellAOR(n0, n1, theta)
			Mj := MacleodMatrixS(lambda, nt.T, n1, theta1)
			Amat = MatMul2C(Amat, Mj)
			theta = theta1
			n0 = n1
		}

		// define eta medium
		eta := n1 * cos0c
		Etam = Vec2C{
			1, eta,
		}
	}

	// Compute the B and C coefficients
	BCVec := MatMulVec2C(Amat, Etam)

	return Macleodr(BCVec, eta0)

}
