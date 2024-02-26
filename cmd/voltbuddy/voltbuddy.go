package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
)

const toleranceFactor = 1.5

var (
	stdinScanner = bufio.NewScanner(os.Stdin)

	e12Resistors = [...]float64{
		1.0, 1.2, 1.5,
		1.8, 2.2, 2.7,
		3.3, 3.9, 4.7,
		5.6, 6.8, 8.2,
	}

	scaledE12Resistors = scaleValues(e12Resistors[:])
)

func readInput(msg string) (string, error) {
	fmt.Print(msg)

	if !stdinScanner.Scan() {
		return "", fmt.Errorf("failed to read from stdin: %w", stdinScanner.Err())
	}

	return stdinScanner.Text(), nil
}

func readFloat(msg string) (float64, error) {
	s, err := readInput(msg)
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.New("not a valid number")
	}

	return i, nil
}

func color(msg, colorCode string) string {
	return "\033[" + colorCode + msg + "\033[0m"
}

func Magenta(msg string) string {
	return color(msg, "95m")
}

func Green(msg string) string {
	return color(msg, "92m")
}

func scaleValues(values []float64) []float64 {
	result := []float64{}
	for _, value := range values {
		result = append(result, value)
		result = append(result, value/10)
		result = append(result, value*10)
	}

	return result
}

func normalizeNumber(x float64) (result float64, exponent uint) {
	exponent = uint(math.Floor(math.Log10(x)))
	result = x / math.Pow10(int(exponent))
	return
}

func findE12Values(r float64) []float64 {
	value, exponent := normalizeNumber(r)

	minValue := value / toleranceFactor
	maxValue := value * toleranceFactor

	result := []float64{}
	for _, e12Value := range scaledE12Resistors {
		if e12Value > minValue && e12Value < maxValue {
			scaledValue := math.Round(e12Value * math.Pow10(int(exponent)))
			result = append(result, scaledValue)
		}
	}

	return result
}

func calcPiPadResistors(zo, attenuation float64) (r1, r2 float64) {
	k := math.Pow(10, attenuation/20.0)
	r1 = (zo / 2) * ((k*k - 1) / k)
	r2 = zo * ((k + 1) / (k - 1))

	return
}

func calcPiPadAttenuation(zo, r1, r2 float64) (attenuation, returnLoss float64) {
	r2p := (r2 * zo) / (r2 + zo)
	zl := (r2 * (r2p + r1)) / (r2 + (r2p + r1))
	rl := (zl - zo) / (zl + zo)
	returnLoss = 20 * math.Log10(math.Abs(rl))

	pIn := 1.0
	vSource := 2 * math.Sqrt(zo*pIn)
	iIn := vSource / (zo + zl)
	vIn := vSource - (iIn * zo)
	vOut := vIn * (r2p / (r2p + r1))
	pOut := vOut * vOut / zo

	gain := 10 * math.Log10(pOut/pIn)
	attenuation = -1 * gain

	return
}

type PiPadResult struct {
	R1          float64
	R2          float64
	Attenuation float64
	ReturnLoss  float64
}

func (r *PiPadResult) String() string {
	return fmt.Sprintf("R1 = %f, R2 = %f, Attenuation = %f, ReturnLoss = %f",
		r.R1, r.R2, r.Attenuation, r.ReturnLoss)
}

func calcPossiblePiPadCombinations(zo, r1, r2 float64) []PiPadResult {
	possibleR1s := findE12Values(r1)
	possibleR2s := findE12Values(r2)

	results := []PiPadResult{}
	for _, r1 := range possibleR1s {
		for _, r2 := range possibleR2s {
			attenuation, returnLoss := calcPiPadAttenuation(zo, r1, r2)
			result := PiPadResult{
				R1:          r1,
				R2:          r2,
				Attenuation: attenuation,
				ReturnLoss:  returnLoss,
			}
			results = append(results, result)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ReturnLoss < results[j].ReturnLoss
	})

	return results[:5]
}

func main() {
	fmt.Println(Magenta("\nPi-Pad Calculator"))
	fmt.Println(`
       r1
 ──┰──████──┰──
   │        │
   █        █
r2 █        █ r2
   █        █
   │        │`)

	zo, err := readFloat("\nImpedance [Ω] = ")
	if err != nil {
		panic(err)
	}

	attenuation, err := readFloat("Attenuation [dB] = ")
	if err != nil {
		panic(err)
	}

	r1, r2 := calcPiPadResistors(zo, attenuation)
	attenuation, returnLoss := calcPiPadAttenuation(zo, r1, r2)
	result := PiPadResult{
		R1:          r1,
		R2:          r2,
		Attenuation: attenuation,
		ReturnLoss:  returnLoss,
	}
	fmt.Println(Magenta("\nIdeal values:"))
	fmt.Println(Green(result.String()))

	fmt.Println(Magenta("\nTop 5 possible E12 combinations:"))
	results := calcPossiblePiPadCombinations(zo, r1, r2)
	for i, result := range results {
		output := fmt.Sprintf("R1 = %f, R2 = %f, Attenuation = %f, ReturnLoss = %f\n",
			result.R1, result.R2, result.Attenuation, result.ReturnLoss)

		if i == 0 {
			output = Green(output)
		}
		fmt.Print(output)
	}
}

// TODO: Add conversion from farad to ampere
// TODO: Add calculation for LED serial resistor
