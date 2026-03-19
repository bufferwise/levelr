package drops

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
)

// Question represents a math MCQ drop question.
type Question struct {
	LaTeX   string    // LaTeX formula for the question (rendered as image)
	Text    string    // plain-text fallback
	Options [4]string // A, B, C, D
	Answer  int       // 0=A, 1=B, 2=C, 3=D
}

// AnswerLabel returns the letter for an answer index.
func AnswerLabel(i int) string {
	return string(rune('A' + i))
}

// pool holds all static questions. Each generator function returns a Question.
var generators = []func() Question{
	genQuadratic,
	genTrigValue,
	genDerivativePolynomial,
	genIntegralPolynomial,
	genExponent,
	genLogarithm,
	genArithmeticSeries,
	genGeometricSeries,
	genQuadraticDiscriminant,
	genTrigIdentity,
	genFactorial,
	genBinomialCoeff,
	genAbsoluteValue,
	genFloorCeil,
	genModular,
	genSqrt,
}

// Random returns a random math question.
func Random() Question {
	gen := generators[rand.IntN(len(generators))]
	return gen()
}

// --- Generators ---

func genQuadratic() Question {
	// Solve x² + bx + c = 0 where roots are integers r1, r2
	r1 := rand.IntN(9) - 4 // -4 to 4
	r2 := rand.IntN(9) - 4
	b := -(r1 + r2)
	c := r1 * r2
	latex := fmt.Sprintf(`x^2 %+dx %+d = 0`, b, c)
	text := fmt.Sprintf("Solve: x² %+dx %+d = 0. Find the sum of roots.", b, c)

	correct := r1 + r2
	return makeMCQ(
		latex,
		text,
		fmt.Sprintf("\\text{Sum of roots} = %d", correct),
		correct,
	)
}

func genTrigValue() Question {
	type trigCase struct {
		angle  int
		fn     string
		latex  string
		value  string
		numVal float64
	}
	cases := []trigCase{
		{30, "sin", `\sin(30^\circ)`, `\frac{1}{2}`, 0.5},
		{60, "sin", `\sin(60^\circ)`, `\frac{\sqrt{3}}{2}`, 0.866},
		{45, "sin", `\sin(45^\circ)`, `\frac{\sqrt{2}}{2}`, 0.707},
		{30, "cos", `\cos(30^\circ)`, `\frac{\sqrt{3}}{2}`, 0.866},
		{60, "cos", `\cos(60^\circ)`, `\frac{1}{2}`, 0.5},
		{45, "tan", `\tan(45^\circ)`, `1`, 1.0},
		{60, "tan", `\tan(60^\circ)`, `\sqrt{3}`, 1.732},
		{30, "tan", `\tan(30^\circ)`, `\frac{1}{\sqrt{3}}`, 0.577},
	}
	tc := cases[rand.IntN(len(cases))]
	return makeLatexMCQ(
		fmt.Sprintf(`\text{Evaluate: } %s`, tc.latex),
		fmt.Sprintf("Evaluate: %s(%d°)", tc.fn, tc.angle),
		tc.value,
		tc.numVal,
	)
}

func genDerivativePolynomial() Question {
	// d/dx (ax^n) = nax^(n-1)
	a := rand.IntN(8) + 2 // 2-9
	n := rand.IntN(4) + 2 // 2-5
	da := n * a
	dn := n - 1
	latex := fmt.Sprintf(`\frac{d}{dx}\left(%dx^{%d}\right)`, a, n)
	text := fmt.Sprintf("Find d/dx of %dx^%d", a, n)
	correctStr := fmt.Sprintf(`%dx^{%d}`, da, dn)
	return makeMCQStr(latex, text, correctStr, da, dn)
}

func genIntegralPolynomial() Question {
	// ∫ ax^n dx = a/(n+1) * x^(n+1) + C
	n := rand.IntN(4) + 1             // 1-4
	a := (n + 1) * (rand.IntN(4) + 1) // make divisible
	coeff := a / (n + 1)
	newN := n + 1
	latex := fmt.Sprintf(`\int %dx^{%d}\,dx`, a, n)
	text := fmt.Sprintf("Evaluate ∫ %dx^%d dx", a, n)
	correctStr := fmt.Sprintf(`%dx^{%d} + C`, coeff, newN)
	return makeMCQStr(latex, text, correctStr, coeff, newN)
}

func genExponent() Question {
	base := rand.IntN(4) + 2 // 2-5
	exp := rand.IntN(5) + 2  // 2-6
	result := intPow(base, exp)
	latex := fmt.Sprintf(`%d^{%d}`, base, exp)
	text := fmt.Sprintf("Calculate %d^%d", base, exp)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genLogarithm() Question {
	// log_b(b^n) = n
	base := rand.IntN(4) + 2 // 2-5
	n := rand.IntN(5) + 1    // 1-5
	val := intPow(base, n)
	latex := fmt.Sprintf(`\log_{%d}(%d)`, base, val)
	text := fmt.Sprintf("Find log base %d of %d", base, val)
	return makeMCQ(latex, text, fmt.Sprintf("%d", n), n)
}

func genArithmeticSeries() Question {
	a := rand.IntN(10) + 1 // 1-10
	d := rand.IntN(5) + 1  // 1-5
	n := rand.IntN(8) + 5  // 5-12
	sum := n * (2*a + (n-1)*d) / 2
	latex := fmt.Sprintf(`S_{%d} = \sum_{k=0}^{%d}(%d + %dk)`, n, n-1, a, d)
	text := fmt.Sprintf("Find S_%d for arithmetic series: a=%d, d=%d", n, a, d)
	return makeMCQ(latex, text, fmt.Sprintf("%d", sum), sum)
}

func genGeometricSeries() Question {
	a := rand.IntN(5) + 1 // 1-5
	r := 2
	n := rand.IntN(4) + 3 // 3-6
	sum := a * (intPow(r, n) - 1) / (r - 1)
	latex := fmt.Sprintf(`\sum_{k=0}^{%d} %d \cdot %d^k`, n-1, a, r)
	text := fmt.Sprintf("Find sum of geometric series: a=%d, r=%d, n=%d", a, r, n)
	return makeMCQ(latex, text, fmt.Sprintf("%d", sum), sum)
}

func genQuadraticDiscriminant() Question {
	a := rand.IntN(3) + 1
	b := rand.IntN(11) - 5
	c := rand.IntN(11) - 5
	disc := b*b - 4*a*c
	latex := fmt.Sprintf(`\Delta = b^2 - 4ac \text{ for } %dx^2 %+dx %+d`, a, b, c)
	text := fmt.Sprintf("Find discriminant of %dx² %+dx %+d", a, b, c)
	return makeMCQ(latex, text, fmt.Sprintf("%d", disc), disc)
}

func genTrigIdentity() Question {
	// sin²θ + cos²θ = ?
	opts := []struct {
		latex   string
		text    string
		answer  string
		correct int
	}{
		{`\sin^2\theta + \cos^2\theta = \text{?}`, "sin²θ + cos²θ = ?", "1", 1},
		{`\sec^2\theta - \tan^2\theta = \text{?}`, "sec²θ - tan²θ = ?", "1", 1},
		{`\csc^2\theta - \cot^2\theta = \text{?}`, "csc²θ - cot²θ = ?", "1", 1},
	}
	o := opts[rand.IntN(len(opts))]
	return makeMCQ(o.latex, o.text, o.answer, o.correct)
}

func genFactorial() Question {
	n := rand.IntN(7) + 3 // 3-9
	result := factorial(n)
	latex := fmt.Sprintf(`%d!`, n)
	text := fmt.Sprintf("Calculate %d!", n)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genBinomialCoeff() Question {
	n := rand.IntN(6) + 4 // 4-9
	r := rand.IntN(n-1) + 1
	result := factorial(n) / (factorial(r) * factorial(n-r))
	latex := fmt.Sprintf(`\binom{%d}{%d}`, n, r)
	text := fmt.Sprintf("Calculate C(%d,%d)", n, r)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genAbsoluteValue() Question {
	a := rand.IntN(21) - 10
	b := rand.IntN(21) - 10
	result := int(math.Abs(float64(a))) + int(math.Abs(float64(b)))
	latex := fmt.Sprintf(`|%d| + |%d|`, a, b)
	text := fmt.Sprintf("Calculate |%d| + |%d|", a, b)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genFloorCeil() Question {
	nums := []float64{3.7, 4.2, -1.5, 2.9, -3.1, 7.8, 5.5}
	x := nums[rand.IntN(len(nums))]
	result := int(math.Floor(x))
	latex := fmt.Sprintf(`\lfloor %.1f \rfloor`, x)
	text := fmt.Sprintf("Calculate ⌊%.1f⌋", x)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genModular() Question {
	a := rand.IntN(90) + 10 // 10-99
	m := rand.IntN(8) + 3   // 3-10
	result := a % m
	latex := fmt.Sprintf(`%d \mod %d`, a, m)
	text := fmt.Sprintf("Calculate %d mod %d", a, m)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

func genSqrt() Question {
	perfects := []int{4, 9, 16, 25, 36, 49, 64, 81, 100, 121, 144, 169, 196, 225}
	n := perfects[rand.IntN(len(perfects))]
	result := int(math.Sqrt(float64(n)))
	latex := fmt.Sprintf(`\sqrt{%d}`, n)
	text := fmt.Sprintf("Calculate √%d", n)
	return makeMCQ(latex, text, fmt.Sprintf("%d", result), result)
}

// --- Helpers ---

func intPow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}

// makeMCQ creates a question with numeric answer and random distractors.
func makeMCQ(latex, text, _ string, correct int) Question {
	q := Question{
		LaTeX:  latex,
		Text:   text,
		Answer: 0, // will be set after shuffle
	}

	// Generate 3 unique distractors
	distractors := make(map[int]bool)
	distractors[correct] = true
	var wrong []int
	for len(wrong) < 3 {
		// Generate distractor near the correct answer
		offset := rand.IntN(11) - 5
		if offset == 0 {
			offset = rand.IntN(5) + 1
		}
		d := correct + offset
		if !distractors[d] {
			distractors[d] = true
			wrong = append(wrong, d)
		}
	}

	// Place answer at random position
	q.Answer = rand.IntN(4)
	wi := 0
	for i := 0; i < 4; i++ {
		if i == q.Answer {
			q.Options[i] = fmt.Sprintf("%d", correct)
		} else {
			q.Options[i] = fmt.Sprintf("%d", wrong[wi])
			wi++
		}
	}

	return q
}

// makeLatexMCQ creates a question with LaTeX answer and float-based distractors.
func makeLatexMCQ(latex, text, answerLatex string, correct float64) Question {
	q := Question{
		LaTeX:  latex,
		Text:   text,
		Answer: 0,
	}

	options := []string{answerLatex}
	used := map[string]bool{answerLatex: true}

	// Simple distractors for trig values
	candidates := []string{
		`\frac{1}{2}`, `\frac{\sqrt{3}}{2}`, `\frac{\sqrt{2}}{2}`,
		`1`, `\sqrt{3}`, `\frac{1}{\sqrt{3}}`, `0`, `2`,
		`\frac{\sqrt{2}}{4}`, `\frac{1}{4}`,
	}
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	for _, c := range candidates {
		if len(options) >= 4 {
			break
		}
		if !used[c] {
			options = append(options, c)
			used[c] = true
		}
	}
	for len(options) < 4 {
		options = append(options, fmt.Sprintf("%.3f", correct+float64(rand.IntN(3)+1)*0.1))
	}

	// Shuffle and track answer position
	q.Answer = rand.IntN(4)
	ans := options[0]
	rest := options[1:]
	rand.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })

	wi := 0
	for i := 0; i < 4; i++ {
		if i == q.Answer {
			q.Options[i] = ans
		} else {
			q.Options[i] = rest[wi]
			wi++
		}
	}

	return q
}

// makeMCQStr creates questions for derivative/integral with coefficient and exponent distractors.
func makeMCQStr(latex, text, correctStr string, coeff, exp int) Question {
	q := Question{
		LaTeX:  latex,
		Text:   text,
		Answer: 0,
	}

	type ce struct{ c, e int }
	correct := ce{coeff, exp}
	used := map[ce]bool{correct: true}
	var wrong []ce
	for len(wrong) < 3 {
		dc := rand.IntN(5) - 2
		de := rand.IntN(3) - 1
		d := ce{coeff + dc, exp + de}
		if d.c == 0 {
			d.c = 1
		}
		if !used[d] {
			used[d] = true
			wrong = append(wrong, d)
		}
	}

	q.Answer = rand.IntN(4)
	wi := 0
	for i := 0; i < 4; i++ {
		if i == q.Answer {
			q.Options[i] = correctStr
		} else {
			w := wrong[wi]
			wi++
			if exp > 0 {
				q.Options[i] = fmt.Sprintf(`%dx^{%d} + C`, w.c, w.e)
			} else {
				q.Options[i] = fmt.Sprintf(`%dx^{%d}`, w.c, w.e)
			}
		}
	}

	return q
}

// LaTeXImageURL returns a URL that renders the given LaTeX as a PNG image.
func LaTeXImageURL(latex string) string {
	// Encode to prevent '+' turning into a space, and URL parsing fails on Discord's proxy bounds
	encoded := url.PathEscape(latex)
	// Use CodeCogs API for LaTeX → crisp black text on white background with padding
	return fmt.Sprintf(
		"https://latex.codecogs.com/png.image?\\dpi{300}\\bg{ffffff}\\fg{000000}\\LARGE\\;%s\\;",
		encoded,
	)
}
