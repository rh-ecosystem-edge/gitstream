package gitutils

//func TestNewRepoWrapper(t *testing.T) {
//	rw := NewRepoWrapper(nil, logr.Discard(), "", "")
//	assert.NotNil(t, rw)
//}
//
//func TestRepoWrapper_CherryPickNoCommit(t *testing.T) {
//	const sha = "e3229f3c533ed51070beff092e5c7694a8ee81f0"
//
//	t.Run("git returns an error", func(t *testing.T) {
//		const (
//			output = "some-output"
//			ret    = 123
//		)
//
//		tempDir := t.TempDir()
//
//		mockExecutor := func(ctx context.Context, bin string, args ...string) *exec.Cmd {
//			helperArgs := append([]string{"-test.run=TestCherryPickHelper", "--", bin}, args...)
//
//			cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
//			cmd.Env = []string{
//				"_TEST_HELPER_PROCESS=1",
//				"_TEST_EXPECTED_DIR=" + tempDir,
//				"_TEST_EXPECTED_COMMAND_LINE=cherry-pick -n " + tempDir,
//				"_TEST_OUTPUT=" + output,
//				"_TEST_RETCODE=" + strconv.Itoa(ret),
//			}
//
//			return cmd
//		}
//
//		rw := NewRepoWrapper(nil, logr.Discard(), tempDir, "")
//		rw.executor = mockExecutor
//
//		err := rw.CherryPickNoCommit(context.Background(), sha)
//		require.Error(t, err)
//
//		pe := &process.Error{}
//
//		assert.ErrorAs(t, err, &pe)
//		assert.Equal(t, ret, pe.ExitCode())
//		assert.Equal(t, []byte(output), pe.Combined())
//	})
//
//	t.Run("git returns no error", func(t *testing.T) {
//		tempDir := t.TempDir()
//
//		mockExecutor := func(ctx context.Context, bin string, args ...string) *exec.Cmd {
//			helperArgs := append([]string{"-test.run=TestCherryPickHelper", "--", bin}, args...)
//
//			cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
//			cmd.Env = []string{
//				"_TEST_HELPER_PROCESS=1",
//				"_TEST_EXPECTED_DIR=" + tempDir,
//				"_TEST_EXPECTED_COMMAND_LINE=cherry-pick -n " + tempDir,
//				"_TEST_OUTPUT=some-output",
//				"_TEST_RETCODE=0",
//			}
//
//			return cmd
//		}
//
//		rw := NewRepoWrapper(nil, logr.Discard(), tempDir, "")
//		rw.executor = mockExecutor
//
//		err := rw.CherryPickNoCommit(context.Background(), sha)
//		assert.NoError(t, err)
//	})
//}
//
//func TestCherryPickHelper(t *testing.T) {
//	if os.Getenv("_TEST_HELPER_PROCESS") != "1" {
//		return
//	}
//
//	assert.Equal(
//		t,
//		strings.Join(os.Args[1:], " "),
//		os.Getenv("_TEST_EXPECTED_COMMAND_LINE"),
//	)
//
//	wd, err := os.Getwd()
//	require.NoError(t, err)
//	assert.Equal(t, os.Getenv("_TEST_EXPECTED_DIR"), wd)
//
//	retCode, err := strconv.Atoi(os.Getenv("_TEST_RETCODE"))
//	require.NoError(t, err)
//
//	fmt.Fprintf(os.Stdout, os.Getenv("_TEST_OUTPUT"))
//	os.Exit(retCode)
//}
