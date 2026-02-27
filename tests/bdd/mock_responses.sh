#!/bin/bash
# Mock AI responses for BDD testing
# This file defines all mock responses for different scenarios

# Get the response based on scenario and context
get_mock_response() {
    local input="$1"
    local scenario_type=""

    # Detect scenario type from input
    if echo "$input" | grep -qi "calculator\|addition\|add"; then
        scenario_type="calculator"
    elif echo "$input" | grep -qi "hello"; then
        scenario_type="hello_world"
    fi

    # Detect request type from input
    if echo "$input" | grep -qi "# Research Topic:"; then
        get_research_response "$scenario_type"
    elif echo "$input" | grep -qi "# Plan Module:"; then
        get_plan_response "$scenario_type"
    elif echo "$input" | grep -qi "task\|doing\|implement"; then
        get_doing_response "$scenario_type"
    else
        # Default: return a generic response
        echo "# Mock Response"
        echo ""
        echo "This is a mock AI response."
    fi
}

# Research response for calculator scenario
get_research_response() {
    local scenario="$1"

    if [ "$scenario" = "calculator" ]; then
        cat <<'EOF'
# Research: Python Calculator Implementation

## Overview

A calculator application in Python that performs basic arithmetic operations including addition, subtraction, multiplication, and division.

## Requirements

- Python 3.x
- Basic arithmetic operations (add, subtract, multiply, divide)
- Clean function-based design
- Error handling for division by zero

## Implementation Strategy

### Core Functions

1. **Addition Function**: Implement `add(a, b)` that returns the sum of two numbers
2. **Subtraction Function**: Implement `subtract(a, b)` that returns the difference
3. **Multiplication Function**: Implement `multiply(a, b)` that returns the product
4. **Division Function**: Implement `divide(a, b)` with zero-check

### Testing Strategy

- Unit tests for each function
- Edge case testing (zero, negative numbers)
- Integration test for calculator workflow

## References

- Python arithmetic operators: +, -, *, /
- Python functions: def keyword
- Error handling: try/except blocks
EOF
    elif [ "$scenario" = "hello_world" ]; then
        cat <<'EOF'
# Research: Hello World Program

## Overview

A simple Hello World program in Python that prints a greeting message to the console.

## Requirements

- Python 3.x
- Print "Hello World" to stdout

## Implementation

Create a Python script that uses the print() function to output "Hello World".

## References

- Python print() function
- Basic Python syntax
EOF
    else
        echo "# Research: Generic Project"
        echo ""
        echo "## Overview"
        echo "This is a generic research document."
    fi
}

# Plan response for calculator scenario
get_plan_response() {
    local scenario="$1"

    if [ "$scenario" = "calculator" ]; then
        cat <<'EOF'
# Plan: Python Calculator Implementation

## 模块概述

**模块职责**: Implement basic arithmetic operations for the calculator

**对应 Research**: None

**依赖模块**: None

**被依赖模块**: None

---

### Job 1: Implement Addition Function

**目标**: Create the add() function that sums two numbers

**前置条件**: None

**Tasks (Todo 列表)**:
- [ ] Task 1: Create calculator.py file
- [ ] Task 2: Define add(a, b) function
- [ ] Task 3: Implement addition logic
- [ ] Task 4: Add docstring
- [ ] Task 5: Test with sample inputs

**验证器**:
```
The add function should:
- Accept two numeric parameters
- Return their sum
- Handle integers and floats
```

---

### Job 2: Implement Subtraction Function

**目标**: Create the subtract() function

**前置条件**: Job 1 completed

**Tasks (Todo 列表)**:
- [ ] Task 1: Define subtract(a, b) function
- [ ] Task 2: Implement subtraction logic
- [ ] Task 3: Add docstring
- [ ] Task 4: Test with sample inputs

**验证器**:
```
The subtract function should return the difference of two numbers.
```

---

### Job 3: Add Main Function

**目标**: Create a main() function to demonstrate usage

**前置条件**: Jobs 1-2 completed

**Tasks (Todo 列表)**:
- [ ] Task 1: Define main() function
- [ ] Task 2: Add example calculations
- [ ] Task 3: Add print statements
- [ ] Task 4: Add if __name__ == "__main__" block

**验证器**:
```
The main function should demonstrate all calculator operations.
```
EOF
    elif [ "$scenario" = "hello_world" ]; then
        cat <<'EOF'
# Plan: Hello World Program

## 模块概述

**模块职责**: Create a simple Hello World program

**对应 Research**: None

**依赖模块**: None

**被依赖模块**: None

---

### Job 1: Create Hello World Script

**目标**: Write a Python script that prints "Hello World"

**前置条件**: None

**Tasks (Todo 列表)**:
- [ ] Task 1: Create hello.py file
- [ ] Task 2: Add print("Hello World") statement
- [ ] Task 3: Test the script

**验证器**:
```
The script should:
- Be a valid Python file
- Print "Hello World" when executed
```
EOF
    else
        echo "# Plan: Generic Project"
        echo ""
        echo "## 模块概述"
        echo ""
        echo "**模块职责**: Generic implementation"
        echo ""
        echo "### Job 1: Generic Task"
        echo ""
        echo "**Tasks (Todo 列表)**:"
        echo "- [ ] Task 1: Do something"
    fi
}

# Doing response (code generation) for calculator scenario
get_doing_response() {
    local scenario="$1"

    if [ "$scenario" = "calculator" ]; then
        cat <<'EOF'
def add(a, b):
    """Add two numbers and return the result."""
    return a + b

def subtract(a, b):
    """Subtract b from a and return the result."""
    return a - b

def multiply(a, b):
    """Multiply two numbers and return the result."""
    return a * b

def divide(a, b):
    """Divide a by b and return the result."""
    if b == 0:
        raise ValueError("Cannot divide by zero")
    return a / b

def main():
    """Demonstrate calculator operations."""
    print("Calculator Demo")
    print("=" * 40)

    # Addition
    result = add(10, 5)
    print(f"10 + 5 = {result}")

    # Subtraction
    result = subtract(10, 5)
    print(f"10 - 5 = {result}")

    # Multiplication
    result = multiply(10, 5)
    print(f"10 * 5 = {result}")

    # Division
    result = divide(10, 5)
    print(f"10 / 5 = {result}")

if __name__ == "__main__":
    main()
EOF
    elif [ "$scenario" = "hello_world" ]; then
        cat <<'EOF'
print("Hello World")
EOF
    else
        echo "# Generic code output"
        echo "print('Hello')"
    fi
}

# Export functions for use in mock_claude.sh
export -f get_mock_response
export -f get_research_response
export -f get_plan_response
export -f get_doing_response
