[![GoDoc](https://godoc.org/github.com/fadion/aria?status.svg)](https://godoc.org/github.com/fadion/aria)
[![Go Report Card](https://goreportcard.com/badge/github.com/fadion/aria)](https://goreportcard.com/report/github.com/fadion/aria)

# Aria Language

Aria is an expressive, interpreted, toy language built as an exercise on designing and interpreting a programming language. It has a noiseless syntax, free of useless semi colons, braces or parantheses, and treats everything as an expression. Technically, it's built with a hand written lexer and parser, a recursive decent one (Pratt), and a tree-walk interpreter. I have never set any goals for it to be either fast, nor bulletproof, so don't expect neither of them.

```swift
let name = "aria language"
let expressive? = fn x
  if x != ""
    return "expressive " + x
  end
  return "sorry, what?"
end

let pipe = name |> expressive?() |> String.capitalize()
IO.puts(pipe) // "Expressive Aria Language"
```

## Usage

If you want to play with the language, but have no interest in toying with its code, you can download a built binary for your operating system. Just head to the [latest release](https://github.com/fadion/aria/releases/latest) and download one of the archives.

The other option, where you get to play with the code and run your changes, is to `go get github.com/fadion/aria` and install it as a local binary with `go install`. Obviously, you'll need `GOROOT` in your path, but I guess you already know what you're doing.

### Run a source file

To run an Aria source file, give it a path relative to the current directory.

```
aria run path/to/file.ari
```

### REPL

As any serious language, Aria provides a REPL too:

```
aria repl
```

## Basic Syntax

As you'd expect, there's a way to declaring variables:

```swift
let name = "John"
let age = 40
```

Once declared, a variable is locked to that value and can't be changed. You guessed it right, they're immutable! We could argue all day, but immutability advocates for safier code. It isn't that hard to pass a modified value to a new variable, isn't it?

Variables have to start with an alphabetic character and then continue either with alphanumeric, underscores, question mark or exclamation mark.

As anything is an expression, except for variable declaration, there are some pretty funny consequences. Everything can be passed to a variable as a value, even block statements like Ifs or Fors:

```swift
let old = if age > 40
  true
else
  false
end
```

Sometimes it's even nicer to inline the If completely, something you can do with almost every block expression. I'm not sure if that's actually readable for anyone, but it's an option:

```swift
let old = if age > 40 then true else false end
```

You've noticed there are no semi colons, braces or stuff like that? To me, it makes for code that's easier to read and scan. Don't confuse it with languages like Python however; in here, whitespace has absolutely no importance. Blocks of code are either inferred where they start, or delimited with keywords like `do`, `then` and `end`.

## Data Types

Aria supports 6 data types: `String`, `Integer`, `Float`, `Boolean`, `Array` and `Dictionary`.

### String

Strings are UTF-8 encoded, meaning that you can stuff in there anything, even emojis.

```swift
let weather = "Hot"
let code = "if\nthen\t\"yes\""
let price = "円500"
let concat = "Hello" + " " + "World"
let subscript = "aname"[2]
```

String concatenation is handled with the `+` operator but trying to concat a string with some other data type will result in a runtime error. Additionally, strings are treated as enumerables. They support subscript, iteration in `for in` loops and most of the array functions.

For the sake of it, there are some escape sequences too: \n, \t, \r, \a, \b, \f and \v. I'm sure you can figure out by yourself what every of them does.

### Integer & Float

Integers and Floats use mostly the same operators, with some minor differences. They can be used in the same expression, for example: 3 + 0.2, where the result is always cast to a Float.

Integers can be represented also as: binary with the 0b prefix, hexadecimal with the 0x prefix and octal with the 0o prefix. They'll be checked for validity at runtime.

```swift
let dec = 27
let oct = 0o33
let hex = 0x1B
let bin = 0b11011
let big = 27_000_000
let arch = 2 ** 32
let ratio = 1.61
let pi = 3.14_159_265
let sci = 0.1e3
let negsci = 25e-5
```

### Boolean

Just `true` or `false`, nothing else!

```swift
let mad = true
let genius = false
```

### Array

Arrays are ordered collections of any data types. You can mix and match strings with integers, or floats with other arrays.
 
 ```swift
 let multi = [5, "Hi", ["Hello", "World"]]
 let names = ["John", "Ben", 1337]
 let john = names[0]
 let concat = ["an", "array"] + ["and", "another"]
 let compare = [1, 2] == [1, 2]
 let nocomma = [5 7 9 "Hi"]
 ```
 
They support subscript with a 0-based index, combining with the `+` operator and comparison with `==` and `!=` that checks every element of both arrays for equality. Obviously, they're enumerables that can be used in `for in` loops and enumerable functions.
 
### Dictionary
 
Dictionaries are hashes with a forced string key and a value of any data type. Unlike arrays, internally their order is irrelevant, so you can't rely on index-based subscripting. They only support key-based subscripting.
 
```swift
let user = ["name": "John", "age": 40]
user["name"]
```

Just to be clear, keys should be string only. Other data types, at least for the moment, are not supported.

## Operators

You can't expect to run some calculations without a good batch of operators, right? Well, Aria has a good range of arithmetic, boolean and bitwise operators to match your needs.

By order of precedence:

```
Pipe: |>
Boolean: && || (AND, OR)
Bitwise: & | ~ (Bitwise AND, OR, NOT)
Equality: == != (Equal, Not equal)
Comparison: < <= > >=
Range: ..
Bitshift: << >> (Bitshift left and right)
Arithmetic: + - * / % ** (addition, substraction, multiplication, division, modulo, power)
```

Not all operators will work with any data type and I'm sure you don't expect that. I've touched on some of them for the special cases, like the `+` for string concatenation or array combining. I'm sure you'll figure them out.

## Functions

Aria treats functions as first class, like any sane language should. It checks all the boxes: they can be passed to variables, as arguments to other functions, and as elements to data structures. The only thing missing for the moment are closures, meaning that a function within a function can't access the parent's variables. This doesn't allow for some interesting techniques like currying, but I'm working on it.

```swift
let add = fn x, y
  x + y
end
```

I've omitted the parantheses too! Of course, you can write the function as `fn (x, y)`, but where's the beauty in that? Calling the function needs the parantheses though:

```swift
let sum = add(1335, 2)
```

Notice the lack of a `return` statement. Functions are expressions, so the last line is considered its return value. In most cases, especially with small functions, you don't have to bother with `return`. However, there are scenarios with multiple return points that need to explicitly tell the interpreter what to return. Let's see the classical factorial example, which is a double win as it shows recursion too.

```swift
let fac = fn n
  if n == 0
    return 1
  end
  
  n * fac(n - 1)
end
``` 

The last statement doesn't need a `return`, as it's the last line and will be automatically inferred. The `if`, on the other hand, is not, so it needs an explicit `return`. Hope it makes sense.

If you're into this kind of things, functions can self-execute:

```swift
let pow_2 = fn x
  x ** 2
end(2)
```

And even passed as elements into data structures:

```swift
let add = fn x, y do x + y end
let list = [1, 2, add]
list[2](5, 7) 
```

## Conditionals

Aria provides two types of conditional expressions: 1) An `if/else` that doesn't support multiple `else if` statements and that's good for simple checks, and 2) A `switch` for anything else. Every block of conditional code has it's own scope, like any other block in Aria; meaning that it can access the previously declared variables, but anything declared inside of them doesn't persist to the rest of the code.

An `if` is pretty simple:

```swift
if 1 == 1
  IO.puts("YES!")
end
```

With the ever present `else` block:

```swift
if 1 == 2
  IO.puts("Not calling me.")
else
  IO.puts("1 isn't equal to 2. Duh!")
end
```

`Switch` expressions on the other hand are more interesting. They can have multiple cases with multiple conditions that break automatically on each successful case. When was the last time you didn't need to break? Exactly!

```swift
let a = 5
switch a
case 2, 3
  IO.puts("Is it 2 or 3?")
case 5
  IO.puts("It is 5. Magic!")
default
  IO.puts("No idea, sorry.")
end
```

Not only that, but a `switch` can behave as a typical if/else when no control condition is provided. It basically becomes a `switch true`.

```swift
let a = "John"
switch
case a == "John"
  IO.puts("John")
case a == "Ben"
  IO.puts("Ben") 
default
  IO.puts("Nobody")
end
```

## For Loop

There's an abundance of `for` loop variations around so Aria takes the short way: a single `for in` loop that's useful to iterate arrays, strings or dictionaries, but that does nothing else.

```swift
for v in [1, 2, 3, 4]
  IO.puts(v)
end
```

Obviously, the result of the loop can be pass to a variable, and that's what makes them interesting to manipulate enumerables.

```swift
let plus_one = for v in [1, 2, 3, 4]
  v + 1
end
```

Passing two arguments for arrays or strings will return the current index and value. For dictionaries, the first argument will be the key.

```swift
for i, v in "abcd"
  IO.puts(i + ":" + v)
end
```

```swift
for k, v in ["name": "John", "age": 40]
  IO.puts(k)
  IO.puts(v)
end
```

## Range Operator

The range operator is a special type of sugar to generate an array of integers or strings. Without a flexible `for` loop, it surely comes in handy.

```swift
let numbers = 0..9
let huge = 999..100
let alphabet = "a".."z"
```

More interesting is using them in a `for in` loop:

```swift
for v in 10..20
  IO.puts(v)
end
```

## Pipe Operator

The pipe operator, inspired by [Elixir](https://elixir-lang.org/), is a very expressive way of chaining functions calls. Instead of very unreadable code like the one below:

```swift
subtract(pow(add(2, 1)))
```

You'll be writing beauties like this one:

```swift
add(2, 1) |> pow() |> substract()
```

The pipe starts from left to right, evaluating each left expression and passing it automatically as the first parameter to the function on the right side. Basically, the result of `add` is passed to `pow`, and finally the result of `pow` to `substract`.

It gets even more interesting when combined with standard library's functions:

```swift
["hello", "world"] |> String.join(" ") |> String.capitalize()
```

Enumerable functions too:

```swift
Enum.map([1, 2, 3], fn x do x + 1 end) |> Enum.filter(fn x do x % 2 == 1 end)

// or even nicer

[1, 2, 3] |> Enum.map(fn x do x + 1 end) |> Enum.filter(fn x do x % 2 == 1 end)
```

The only gotcha for the moment is that piped expressions can't span multiple lines, but it's something I'm looking into.

## Immutability

Now that you've seen most of the language constructs, it's time to fight the dragon. Enforced immutability is something you may not agree with immediately, but it makes a lot of sense the more you think about it. What you'll win is increased clarity and programs that are easier to reason about.

In Aria, this won't work:

```swift
let a = 10
a = 15 // Parse error: Unexpected expression '='
```

It won't even parse, as assignement is allowed only in let statements, but not as an expression. What to do then? Very easy, just declare a new variable!

Iterators are typical examples where mutability is seeked for. The dreaded `i` variable shows itself in almost every language's `for` loop. Aria keeps it simple with the `for in` loop that tracks the index and value. Even if it looks like it, the index and value aren't mutable values, but arguments to each iteration of the loop.

```swift
let numbers = [10, 5, 9]
for k, v in numbers
  IO.puts(v) 
  IO.puts(numbers[k] // same thing
end
```

But there may be more complicated scenarios, like wanting to modify an array's values. Sure, you can do it with the `for in` loop, but higher order functions play even better:

```swift
let plus_one = Enum.map([1, 2, 3], fn x
  x + 1
end)
IO.puts(plus_one) // [2, 3, 4]
```

Filter is also useful to "clean" an array of unwanted values:

```swift
let even = Enum.filter(1..10, fn x
  x % 2 == 0
end)
IO.puts(even) // [2, 4, 6, 8, 10]
```

What about accumulators? Let's say you want the product of all the elements of an array (factorial) and obviously, you'll need a mutable variable to hold it. That's what `reduce` is for:

```swift
let product = Enum.reduce(1..5, 1, fn x, acc
  x * acc
end)
IO.puts(product)
``` 

All of these functions and others in the standard library can be mixed and matched to your needs. I'm sure you'll find plenty of scenarios where the current capabilities of the language can't hold up to the promise and fail to achieve something without mutable values. I'll try and fix those holes!

## Modules

Modules are very simple containers of data and nothing more. They're not an imitation of classes, as they can't be initialized, don't have any type of access control, inheritance or whatever. If you need to think in Object Oriented terms, they're like a class with only static properties and methods. They're good to give some structure to a program, but not to represent cars, trees and cats.

```swift
module Color
  let white = "#fff"
  let grey = "#666"
  let hexToRGB = fn hex
    // some calculations
  end
end

let background = Color.white
let font_color = Color.hexToRGB(Color.grey)
```

There can't be any other statement in modules except `let`, but those variables can have any expression possible. The dot syntax of calling a module property or function may remind you of classes, but still, they're not!

Keep in mind that the Aria interpreter is single pass and as such, it will only recognize calls to a module that has already been declared. 

## Comments

Nothing fancy in here! You can comment your code using both inline or block comments:

```
// an inline comment
/*
  I'm spanning multiple
  lines.
*/
```

## Standard Library

Right now it's a small library of functions, but it's expending continually. Head over to the [documentation](https://github.com/fadion/aria/wiki/Standard-Library).

## Future Plans

Although this is a language made purely for fun and experimentation, it doesn't mean I will abandon it in it's first release. Adding other features means I'll learn even more!

In the near future, hopefully, I plan to:

- Improve the Standard Library with more functions.
- Support closures and ~~recursion~~.
- Add a short syntax for functions in the form of `x -> x`.
- Add importing of other files.
- ~~Add the pipe operator!~~
- Support optional values for null returns.
- Write more tests!
- Write some useful benchmarks with non-trivial programs.

## Credits

Aria was developed by Fadion Dashi, a freelance web and mobile developer from Tirana.

The implementation is based on the fantastic [Writing an Interpreter in Go](https://interpreterbook.com/). If you're even vaguely interested in interpreters, with Golang or not, I highly suggest that book.

The `reader.Buffer` package is a "fork" of Golang's `bytes.Buffer`, in which I had to add a method that reads a rune without moving the internal cursor. I hate doing that, but unfortunately couldn't find a way out of it. That package has its own BSD-style [license](https://github.com/golang/go/blob/master/LICENSE).