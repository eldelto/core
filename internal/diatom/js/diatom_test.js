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

document.addEventListener("DOMContentLoaded", function() {
  const testResults = document.querySelector("#test-results");

  const testData = [
    new TestData("add", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 2, ADD], [7], []),
    new TestData("add bad", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 5, ADD], [7], []),
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