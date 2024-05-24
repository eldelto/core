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
	constructor(name, program, wantDataStack, wantReturnStack, input, wantOutput) {
		this.name = name;
		this.program = new Uint8Array(program);
		this.wantDataStack = wantDataStack;
		this.wantReturnStack = wantReturnStack;
		this.input = input;
		this.wantOutput = wantOutput;
	}
}

async function runTests() {
	const testResults = document.querySelector("#test-results");

	const testData = [
		new TestData("exit", [EXIT], [], [],"",""),
		new TestData("nop", [NOP], [], [],"",""),
		new TestData("ret", [CONST, 0, 0, 0, 8, RPUT, RET, EXIT, CONST, 0, 0, 0, 11], [11], [],"",""),
		new TestData("const", [CONST, 0, 0, 0, 11], [11], [],"",""),
		new TestData("fetch", [CONST, 0, 0, 0, 7, FETCH, CONST, 0, 0, 0, 11], [11], [],"",""),
		new TestData("store", [CONST, 0, 0, 0, 11, CONST, 0, 0, 0, 12, FETCH, CONST, 0, 0, 0, 0], [11], [],"",""),
		new TestData("add", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 3, ADD], [8], [],"",""),
		new TestData("subtract", [CONST, 0, 0, 0, 3, CONST, 0, 0, 0, 5, SUBTRACT], [-2], [],"",""),
		new TestData("multiply", [CONST, 0, 0, 0, 3, CONST, 0, 0, 0, 5, MULTIPLY], [15], [],"",""),
		new TestData("divide", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 3, DIVIDE], [2], [],"",""),
		new TestData("modulo", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 3, MODULO], [1], [],"",""),
		new TestData("dup", [CONST, 0, 0, 0, 7, DUP], [7, 7], [],"",""),
		new TestData("drop", [CONST, 0, 0, 0, 7, DUP, DROP], [7], [],"",""),
		new TestData("swap", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 2, SWAP], [2, 7], [],"",""),
		new TestData("over", [CONST, 0, 0, 0, 7, CONST, 0, 0, 0, 2, OVER], [7, 2, 7], [],"",""),
		new TestData("conditional jmp true", [CONST, 255, 255, 255, 255, CJMP, 0, 0, 0, 16, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [11], [],"",""),
		new TestData("conditional jmp false", [CONST, 0, 0, 0, 0, CJMP, 0, 0, 0, 16, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [22], [],"",""),
		new TestData("call without return", [CALL, 0, 0, 0, 11, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [11], [5],"",""),
		new TestData("call with return", [CALL, 0, 0, 0, 11, CONST, 0, 0, 0, 22, EXIT, RET, CONST, 0, 0, 0, 11], [22], [],"",""),
		new TestData("stack call without return", [CONST, 0, 0, 0, 12, SCALL, CONST, 0, 0, 0, 22, EXIT, CONST, 0, 0, 0, 11], [11], [6],"",""),
		new TestData("stack call with return", [CONST, 0, 0, 0, 12, SCALL, CONST, 0, 0, 0, 22, EXIT, RET, CONST, 0, 0, 0, 11], [22], [],"",""),
		new TestData("equals true", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 5, EQUALS], [-1], [],"",""),
		new TestData("equals false", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 4, EQUALS], [0], [],"",""),
		new TestData("not", [CONST, 0, 0, 0, 0, NOT], [-1], [],"",""),
		new TestData("and", [CONST, 0, 0, 0, 3, CONST, 0, 0, 0, 5, AND], [1], [],"",""),
		new TestData("or", [CONST, 0, 0, 0, 1, CONST, 0, 0, 0, 6, OR], [7], [],"",""),
		new TestData("lesser than false", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 5, LT], [0], [],"",""),
		new TestData("lesser than true", [CONST, 0, 0, 0, 4, CONST, 0, 0, 0, 5, LT], [-1], [],"",""),
		new TestData("greater than false", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 5, GT], [0], [],"",""),
		new TestData("greater than true", [CONST, 0, 0, 0, 5, CONST, 0, 0, 0, 4, GT], [-1], [],"",""),
		new TestData("rput", [CONST, 0,0,0,5, RPUT], [], [5],"",""),
		new TestData("rpop", [CONST, 0,0,0,5, RPUT, RPOP], [5], [],"",""),
		new TestData("rpeek", [CONST, 0,0,0,5, RPUT, RPEEK], [5], [5],"",""),
		new TestData("byte fetch", [CONST, 0,0,0,10, BFETCH, EXIT, 0,0,0,5], [5], [],"",""),
		new TestData("byte store", [CONST, 0,0,0,7, CONST, 0,0,0,20, BSTORE, CONST, 0,0,0,20, BFETCH, EXIT, 0,0,0,5], [7], [],"",""),
		new TestData("dump", [CONST, 0,0,0,7, DUMP], [], [],"",""),
		new TestData("key", [KEY], [65], [], "A", ""),
		new TestData("emit", [CONST, 0,0,0,65, EMIT], [], [], "", "A"),
	];

	for (const tt of testData) {
		const row = document.createElement("tr");

		const testName = document.createElement("td");
		testName.textContent = tt.name;
		row.append(testName);

		const testResult = document.createElement("td");
		testResult.textContent = "✅";

		const testError = document.createElement("td");

		const inputElement = document.querySelector("#test-input");
		inputElement.value = tt.input;
		const outputElement = document.querySelector("#test-output");

		const vm = new DiatomVM().withInput("#test-input")
			  .withOutput("#test-output");
		try {
			vm.load(tt.program);
			await vm.execute();

			assertStack(tt.wantDataStack, vm.dataStack, "vm.dataStack");
			assertStack(tt.wantReturnStack, vm.returnStack, "vm.returnStack");
			assertEquals(tt.wantOutput, outputElement.textContent.trim(), "vm output");
		} catch (error) {
			testResult.textContent = "❌";
			testError.textContent = error;
			console.error(error);
		} finally {
			vm.reset();
		}

		row.append(testResult);
		row.append(testError);
		testResults.append(row);
	}

	const vm = new DiatomVM().withInput("#test-input")
		  .withOutput("#test-output");
	await vm.execute();
}

document.addEventListener("DOMContentLoaded", runTests);
