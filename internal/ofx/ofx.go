package ofx

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/eldelto/core/internal/util"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	transactionStart = "<STMTTRN>"
	transactionEnd   = "</STMTTRN>"
	dtPostedStart    = "<DTPOSTED>"
	trnAmtStart      = "<TRNAMT>"
	nameStart        = "<NAME>"
	memoStart        = "<MEMO>"

	DateTimeLayout = "20060102"
)

type Currency int

func (c Currency) String() string {
	return fmt.Sprintf("%d.%d", c/100, util.AbsI(c%100))
}

type Transaction struct {
	DateTime time.Time
	Amount   Currency
	Name     string
	Memo     string
}

func parseDtPosted(t *Transaction, line string) error {
	if !strings.Contains(line, dtPostedStart) {
		return nil
	}

	rawAttribute := strings.ReplaceAll(line, dtPostedStart, "")[:8]
	dateTime, err := time.Parse(DateTimeLayout, rawAttribute)
	if err != nil {
		return fmt.Errorf("%q is not a valid date time", rawAttribute)
	}
	t.DateTime = dateTime

	return nil
}

func parseTrnAmt(t *Transaction, line string) error {
	if !strings.Contains(line, trnAmtStart) {
		return nil
	}

	rawAttribute := strings.ReplaceAll(line, trnAmtStart, "")
	rawIntValue := strings.ReplaceAll(rawAttribute, ".", "")

	value, err := strconv.ParseInt(rawIntValue, 10, 64)
	if err != nil {
		return fmt.Errorf("%q is not a valid transaction amount", rawAttribute)
	}
	t.Amount = Currency(value)

	return nil
}

func parseName(t *Transaction, line string) error {
	if !strings.Contains(line, nameStart) {
		return nil
	}

	rawAttribute := strings.ReplaceAll(line, nameStart, "")
	t.Name = rawAttribute

	return nil
}

func parseMemo(t *Transaction, line string) error {
	if !strings.Contains(line, memoStart) {
		return nil
	}

	rawAttribute := strings.ReplaceAll(line, memoStart, "")
	t.Memo = rawAttribute

	return nil
}

func Parse(r io.Reader) ([]Transaction, error) {
	transactions := []Transaction{}
	var transaction *Transaction

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == transactionStart {
			transaction = &Transaction{}
		} else if line == transactionEnd {
			transactions = append(transactions, *transaction)
		} else {
			if err := parseDtPosted(transaction, line); err != nil {
				return nil, err
			}
			if err := parseTrnAmt(transaction, line); err != nil {
				return nil, err
			}
			if err := parseName(transaction, line); err != nil {
				return nil, err
			}
			if err := parseMemo(transaction, line); err != nil {
				return nil, err
			}
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to parse OFX file: %w", err)
	}

	return transactions, nil
}
