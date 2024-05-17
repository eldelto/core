function assertEquals(expected, actual, title) {
  if (expected !== actual) {
    throw new Error(`${title} should be '${expected}' but was '${actual}'`)
  }
}

function assertStack(expected, stack, title) {
  expected.forEach((value, i) =>
    assertEquals(value, stack.data[i], title + " at index " + i));
}

class TestData {
  constructor(name, program, wantDataStack, wantReturnStack) {
    this.name = name;
    this.program = new Uint8Array(program);
    this.wantDataStack = wantDataStack;
    this.wantReturnStack = wantReturnStack;
  }
}

document.addEventListener("DOMContentLoaded", function () {
  const testResults = document.querySelector("#test-results");

  const testData = [
    new TestData("exit", [EXIT], [], []),
    new TestData("nop", [NOP], [], []),

    //new TestData("const @x rput ret exit :x const 11", [11], []),

    new TestData("const", [CONST, 0, 0, 0, 11], [11], []),
    new TestData("fetch", [CONST, 0, 0, 0, 7, FETCH, CONST, 0, 0, 0, 11], [11], []),
    new TestData("store", [CONST, 0, 0, 0, 11, CONST, 0, 0, 0, 12, FETCH, CONST, 0, 0, 0, 0], [11], []),
    new TestData("add", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 3, ADD], [8], []),
    new TestData("subtract", [CONST, 0, 0, 0, 3, CONST, 0, 0, 0, 5, SUBTRACT], [-2], []),
    new TestData("multiply", [CONST, 0, 0, 0, 3, CONST, 0, 0, 0, 5, MULTIPLY], [15], []),
    new TestData("divide", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 3, DIVIDE], [2], []),
    new TestData("modulo", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 3, MODULO], [1], []),
    new TestData("dup", [CONST, 0, 0, 0, 7, DUP], [7, 7], []),
    new TestData("drop", [CONST, 0, 0, 0, 7, DUP, DROP], [7], []),
    new TestData("swap", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 2, SWAP], [2, 7], []),
    new TestData("over", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 2, OVER], [7, 2, 7], []),


    // TODO: Fix the jump targets
    new TestData("cjmp true", [CONST, 255, 255, 255, 255, CJMP, 0, 0, 0, 2, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [11], []),
    new TestData("cjmp false", [CONST, 0, 0, 0, 0, CJMP, 0, 0, 0, 2, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [22], []),


    new TestData("call @x const 22 exit :x const 11", [11], [5]),
    new TestData("call @x const 22 exit :x ret const 11", [22], []),
    new TestData("const @x scall const 22 exit :x const 11", [11], [6]),
    new TestData("const @x scall const 22 exit :x ret const 11", [22], []),
    new TestData("const 5 const 5 =", [-1], []),
    new TestData("const 5 const 4 =", [0], []),
    new TestData("const 0 ~", [-1], []),
    new TestData("const 3 const 5 &", [1], []),
    new TestData("const 1 const 6 |", [7], []),
    new TestData("const 5 const 5 <", [0], []),
    new TestData("const 4 const 5 <", [-1], []),
    new TestData("const 5 const 5 >", [0], []),
    new TestData("const 5 const 4 >", [-1], []),
    new TestData("const 5 rput", [], [5]),
    new TestData("const 5 rput rpop", [5], []),
    new TestData("const 5 rput rpeek", [5], [5]),
    new TestData("const 10 b@ exit 5", [5], []),
    new TestData("const 7 const 20 b! const 20 b@ exit 5", [7], []),
    new TestData("const 777 const 20 ! const 20 @ exit 5", [777], []),

  ];

  testData.forEach(tt => {
    const row = document.createElement("tr");

    const testName = document.createElement("td");
    testName.textContent = tt.name;
    row.append(testName);

    const testResult = document.createElement("td");
    testResult.textContent = "✅";

    const testError = document.createElement("td");

    try {
      const vm = new DiatomVM();
      vm.load(tt.program);
      vm.execute();

      // TODO: Assertion
      assertStack(tt.wantDataStack, vm.dataStack, "vm.dataStack");
      assertStack(tt.wantReturnStack, vm.returnStack, "vm.returnStack");
    } catch (error) {
      testResult.textContent = "❌";
      testError.textContent = error;
      console.error(error);
    }

    row.append(testResult);
    row.append(testError);
    testResults.append(row);
  })
});