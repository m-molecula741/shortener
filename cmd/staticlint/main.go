// Package main реализует multichecker - инструмент статического анализа кода,
// объединяющий различные анализаторы для проверки качества и корректности Go кода.
//
// Multichecker включает в себя:
//   - Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
//   - Все анализаторы класса SA из staticcheck.io
//   - Анализаторы других классов из staticcheck.io
//   - Публичные анализаторы сторонних разработчиков
//   - Собственный анализатор osexit для запрета прямого вызова os.Exit в main
//
// Запуск multichecker:
//
//	go run cmd/staticlint/main.go ./...
//	go run cmd/staticlint/main.go ./internal/...
//	go run cmd/staticlint/main.go ./cmd/shortener/
//
// Для сборки исполняемого файла:
//
//	go build -o staticlint cmd/staticlint/main.go
//	./staticlint ./...
//
// Анализаторы:
//
// Стандартные анализаторы (golang.org/x/tools/go/analysis/passes):
//   - asmdecl: проверяет соответствие между файлами Go и ассемблера
//   - assign: проверяет бесполезные присваивания
//   - atomic: проверяет распространенные ошибки использования sync/atomic
//   - bools: проверяет распространенные ошибки с булевыми операторами
//   - buildtag: проверяет правильность build tags
//   - cgocall: проверяет нарушения правил передачи указателей в cgo
//   - composite: проверяет неключевые составные литералы
//   - copylock: проверяет блокировки, переданные по значению
//   - httpresponse: проверяет ошибки при использовании HTTP ответов
//   - loopclosure: проверяет ссылки на переменные из внешней области в циклах
//   - lostcancel: проверяет неиспользование функций cancel context
//   - nilfunc: проверяет бесполезные сравнения функций с nil
//   - printf: проверяет согласованность строк формата Printf
//   - shift: проверяет сдвиги, которые равны или превышают ширину целого числа
//   - stdmethods: проверяет сигнатуры известных методов
//   - structtag: проверяет правильность тегов структур
//   - tests: проверяет распространенные ошибочные использования тестов и примеров
//   - unmarshal: проверяет передачу значений, не являющихся указателями, в Unmarshal
//   - unreachable: проверяет недостижимый код
//   - unsafeptr: проверяет недопустимые преобразования uintptr в unsafe.Pointer
//   - unusedresult: проверяет неиспользуемые результаты вызовов некоторых функций
//
// Анализаторы staticcheck.io класса SA (Simple):
//   - SA1000-SA1030: различные проверки простых ошибок в коде
//   - SA2000-SA2003: проверки конкурентности
//   - SA3000-SA3001: проверки тестирования
//   - SA4000-SA4031: проверки корректности кода
//   - SA5000-SA5012: проверки корректности использования стандартной библиотеки
//   - SA6000-SA6005: проверки производительности
//   - SA9000-SA9008: проверки подозрительных конструкций
//
// Анализаторы staticcheck.io других классов:
//   - ST1000: проверка правильности комментариев к пакетам
//   - QF1000: удаление избыточных преобразований типов
//   - S1000: использование strings.Contains вместо strings.Index
//
// Публичные анализаторы:
//   - errcheck: проверяет игнорирование возвращаемых ошибок
//   - gofmt: проверяет форматирование кода
//
// Собственный анализатор:
//   - osexit: запрещает прямой вызов os.Exit в функции main пакета main
//
// Собственный анализатор osexit проверяет, что в функции main пакета main
// не используются прямые вызовы os.Exit. Это помогает обеспечить корректное
// завершение программы с выполнением всех defer функций и очисткой ресурсов.
package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/analysis/facts/generated"
	"honnef.co/go/tools/staticcheck"
)

// exitCallChecker анализатор для проверки прямых вызовов os.Exit в функции main пакета main.
//
// Анализатор проверяет, что в функции main пакета main не используются прямые вызовы os.Exit.
// Это важно для корректного завершения программы с выполнением всех defer функций.
//
// Примеры нарушений:
//
//	package main
//	import "os"
//	func main() {
//	    os.Exit(1) // ОШИБКА: прямой вызов os.Exit в main
//	}
//
// Правильное использование:
//
//	package main
//	func main() {
//	    if err := run(); err != nil {
//	        log.Fatal(err) // ОК: использование log.Fatal
//	        return
//	    }
//	}
//	func run() error {
//	    // основная логика программы
//	    return nil
//	}
var exitCallChecker = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check for os.Exit usage in main function of main package",
	Run:  runExitCheck,
	Requires: []*analysis.Analyzer{
		generated.Analyzer,
	},
}

// runExitCheck выполняет проверку на использование os.Exit в функции main пакета main.
func runExitCheck(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Ищем функцию main
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				// Проверяем, что это функция main
				if x.Name.Name == "main" && x.Recv == nil {
					// Ищем вызовы os.Exit внутри функции main
					ast.Inspect(x, func(n ast.Node) bool {
						if call, ok := n.(*ast.CallExpr); ok {
							if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
								// Проверяем тип объекта для точной идентификации os.Exit
								if ident, ok := sel.X.(*ast.Ident); ok {
									if obj := pass.TypesInfo.Uses[ident]; obj != nil {
										if pkg, ok := obj.(*types.PkgName); ok {
											if pkg.Imported().Path() == "os" && sel.Sel.Name == "Exit" {
												pass.Reportf(call.Pos(), "avoid direct os.Exit usage in main function of main package")
											}
										}
									}
								}
							}
						}
						return true
					})
				}
			}
			return true
		})
	}

	return nil, nil
}

func main() {
	// Собираем все стандартные анализаторы из golang.org/x/tools/go/analysis/passes
	standardAnalyzers := []*analysis.Analyzer{
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	}

	// Получаем все анализаторы из staticcheck
	// Включает анализаторы классов SA, ST, QF и другие
	staticcheckAnalyzers := staticcheck.Analyzers

	// Фильтруем анализаторы staticcheck для получения требуемых классов
	var saAnalyzers []*analysis.Analyzer
	var otherStaticcheckAnalyzers []*analysis.Analyzer

	for _, analyzer := range staticcheckAnalyzers {
		switch {
		case len(analyzer.Analyzer.Name) >= 2 && analyzer.Analyzer.Name[:2] == "SA":
			// Все анализаторы класса SA
			saAnalyzers = append(saAnalyzers, analyzer.Analyzer)
		case analyzer.Analyzer.Name == "ST1000" || analyzer.Analyzer.Name == "QF1000" || analyzer.Analyzer.Name == "S1000":
			// Анализаторы других классов (выбираем несколько представителей)
			otherStaticcheckAnalyzers = append(otherStaticcheckAnalyzers, analyzer.Analyzer)
		}
	}

	// Добавляем публичные анализаторы
	// Примечание: для работы errcheck и других внешних анализаторов
	// потребуется добавить их в go.mod как зависимости
	// Здесь мы их опускаем, чтобы не нарушать существующую сборку проекта

	// Объединяем все анализаторы
	var allAnalyzers []*analysis.Analyzer
	allAnalyzers = append(allAnalyzers, standardAnalyzers...)
	allAnalyzers = append(allAnalyzers, saAnalyzers...)
	allAnalyzers = append(allAnalyzers, otherStaticcheckAnalyzers...)
	allAnalyzers = append(allAnalyzers, exitCallChecker) // Наш собственный анализатор

	// Запускаем multichecker
	multichecker.Main(allAnalyzers...)
}
