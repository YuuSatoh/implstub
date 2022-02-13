package implstub_test

import (
	"implstub"
	"testing"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	// testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	// analysistest.Run(t, testdata, implstub.Generator, "a")
}

func TestExec(t *testing.T) {
	type args struct {
		output          *string
		overwrite       bool
		pointerReciever bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				output:          nil,
				overwrite:       true,
				pointerReciever: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := implstub.Exec(tt.args.output, tt.args.overwrite, tt.args.pointerReciever); (err != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestArrangePackagePath(t *testing.T) {
	type args struct {
		dstFilePath string
		srcFilePath string
		srcParams   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "パラメーターの冗長なパッケージパスが取り除かれる。filePathにファイル名までを指定",
			args: args{
				dstFilePath: "test/model.go",
				srcFilePath: "test/repo.go",
				srcParams:   "(adb implstub/test/a.DB, db implstub/test/b.DB)",
			},
			want: "(adb a.DB, db b.DB)",
		},
		{
			name: "パラメーターの冗長なパッケージパスが取り除かれる。filePathはパッケージパスを指定",
			args: args{
				dstFilePath: "test/model",
				srcFilePath: "test/repo",
				srcParams:   "(adb implstub/test/a.DB, db implstub/test/b.DB)",
			},
			want: "(adb a.DB, db b.DB)",
		},
		{
			name: "command-line-argumentsがパラメーターで指定されている場合はsrcFilePathに置換される",
			args: args{
				dstFilePath: "test/model/a.go",
				srcFilePath: "test/repo/a.go",
				srcParams:   "(adb implstub/test/a.DB, db command-line-arguments.DB)",
			},
			want: "(adb a.DB, db repo.DB)",
		},
		{
			name: "配置先のdstFilePathがパラメーターで指定されている場合取り除かれる",
			args: args{
				dstFilePath: "test/model/a.go",
				srcFilePath: "test/repo/a.go",
				srcParams:   "(adb implstub/test/a.DB, db test/model.DB)",
			},
			want: "(adb a.DB, db DB)",
		},
		{
			name: "変数名が指定されていない場合は記載しない",
			args: args{
				dstFilePath: "test/model",
				srcFilePath: "test/repo",
				srcParams:   "(implstub/test/a.DB, implstub/test/b.DB)",
			},
			want: "(a.DB, b.DB)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := implstub.ArrangePackagePath(tt.args.dstFilePath, tt.args.srcFilePath, tt.args.srcParams); got != tt.want {
				t.Errorf("ArrangePackagePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
