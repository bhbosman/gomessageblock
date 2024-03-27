package gomessageblock_test

import (
	"bytes"
	"crypto/sha1"
	"github.com/bhbosman/gomessageblock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReaderWriter(t *testing.T) {
	t.Run("empty read", func(t *testing.T) {
		rw := gomessageblock.NewReaderWriter()
		read, err := rw.Read([]byte{0})
		require.Error(t, err)
		require.Equal(t, 0, read)
	})

	t.Run("one 4096W, 4 x 1024R, empty buffer", func(t *testing.T) {
		rw := gomessageblock.NewReaderWriter()
		n, err := rw.Write(make([]byte, 4096))
		require.NoError(t, err)
		require.Equal(t, 4096, n)
		readBuffer := make([]byte, 1024)
		for i := 0; i <= 3; i++ {
			n, err = rw.Read(readBuffer)
			require.NoError(t, err)
			require.Equal(t, 1024, n)
		}
		require.Equal(t, 0, rw.Size())
		n, err = rw.Read(readBuffer)
		require.Error(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("one 4096W, 4 x 1024R, empty buffer", func(t *testing.T) {
		rw := gomessageblock.NewReaderWriter()
		for i := 0; i < 8; i++ {
			n, err := rw.Write(make([]byte, 1024))
			require.NoError(t, err)
			require.Equal(t, 1024, n)
		}
		readBuffer := make([]byte, 2048)
		for i := 0; i < 4; i++ {
			n, err := rw.Read(readBuffer)
			require.NoError(t, err)
			require.Equal(t, 2048, n)
		}
		require.Equal(t, 0, rw.Size())
		n, err := rw.Read(readBuffer)
		require.Error(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("", func(t *testing.T) {
		rw := gomessageblock.NewReaderWriterSize(512)
		for i := 0; i < 8; i++ {
			n, err := rw.Write(make([]byte, 1024))
			require.NoError(t, err)
			require.Equal(t, 1024, n)
		}
		rw.Flatten()
	})

	t.Run("flatten to next", func(t *testing.T) {
		rw01 := gomessageblock.NewReaderWriterSize(512)
		_, _ = rw01.Write([]byte{1})
		rw02 := gomessageblock.NewReaderWriterSize(512)
		_, _ = rw02.Write([]byte{2})
		rw01.Add(rw02)
		require.Equal(t, 2, rw01.Size())
		require.Equal(t, 1022, rw01.Waste())
		flatten, _ := rw01.Flatten()
		require.Len(t, flatten, 2)
		require.Equal(t, 2, rw01.Size())
		require.Equal(t, 510, rw01.Waste())
	})

	t.Run("flatten to next", func(t *testing.T) {
		rw01 := gomessageblock.NewReaderWriterSize(128)
		_, _ = rw01.Write(make([]byte, 20))
		_, _ = rw01.Write(make([]byte, 2048))
		require.Equal(t, 2068, rw01.Size())
		require.Equal(t, 108, rw01.Waste())

		rw02 := gomessageblock.NewReaderWriterSize(256)
		_, _ = rw02.Write(make([]byte, 20))
		_, _ = rw02.Write(make([]byte, 4096))
		require.Equal(t, 4116, rw02.Size())
		require.Equal(t, 236, rw02.Waste())

		rw01.Add(rw02)
		require.Equal(t, 4116+2068, rw01.Size())
		require.Equal(t, 108+236, rw01.Waste())
		flatten, _ := rw01.Flatten()
		require.Len(t, flatten, 4116+2068)
		require.Equal(t, 4116+2068, rw01.Size())
		require.Equal(t, 108+236, rw01.Waste())
	})
	t.Run("Add Three Buffers, verify with hash", func(t *testing.T) {
		data001 := []byte("ReaderWriter_test_0001")
		data002 := []byte("ReaderWriter_test_0002")
		data003 := []byte("ReaderWriter_test_0003")
		sha1Hash := sha1.New()
		_, err := sha1Hash.Write(data001)
		require.NoError(t, err)
		_, err = sha1Hash.Write(data002)
		require.NoError(t, err)
		_, err = sha1Hash.Write(data003)
		require.NoError(t, err)

		mustbe := sha1Hash.Sum(nil)
		sha1Hash.Reset()

		rw01 := gomessageblock.NewReaderWriterSize(128)
		_, err = rw01.Write(data001)
		require.NoError(t, err)
		rw02 := gomessageblock.NewReaderWriterSize(128)
		_, err = rw02.Write(data002)
		require.NoError(t, err)

		rw03 := gomessageblock.NewReaderWriterSize(128)
		_, err = rw03.Write(data003)
		require.NoError(t, err)

		err = rw01.Add(rw02, rw03)
		require.NoError(t, err)

		err = rw01.Dump(sha1Hash)
		require.NoError(t, err)

		valueIs := sha1Hash.Sum(nil)
		t.Logf("%v\n", mustbe)
		t.Logf("%v\n", valueIs)
		require.Equal(t, mustbe, valueIs)
	})
	t.Run("Hash", func(t *testing.T) {
		t.Run("EmptyHash", func(t *testing.T) {
			sha1Hash := sha1.New()
			mustbe := sha1Hash.Sum(nil)
			sha1Hash.Reset()
			rw01 := gomessageblock.NewReaderWriterSize(128)
			err := rw01.Dump(sha1Hash)
			require.NoError(t, err)

			valueIs := sha1Hash.Sum(nil)
			t.Logf("%v\n", mustbe)
			t.Logf("%v\n", valueIs)
			require.Equal(t, mustbe, valueIs)
		})

		t.Run("One Buffer", func(t *testing.T) {
			data := []byte("ReaderWriter_test")
			sha1Hash := sha1.New()
			_, err := sha1Hash.Write(data)
			require.NoError(t, err)
			mustbe := sha1Hash.Sum(nil)
			sha1Hash.Reset()

			rw01 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw01.Write(data)
			require.NoError(t, err)
			err = rw01.Dump(sha1Hash)
			require.NoError(t, err)

			valueIs := sha1Hash.Sum(nil)
			t.Logf("%v\n", mustbe)
			t.Logf("%v\n", valueIs)
			require.Equal(t, mustbe, valueIs)
		})

		t.Run("Two Buffers", func(t *testing.T) {
			data001 := []byte("ReaderWriter_test_0001")
			data002 := []byte("ReaderWriter_test_0002")
			sha1Hash := sha1.New()
			_, err := sha1Hash.Write(data001)
			require.NoError(t, err)
			_, err = sha1Hash.Write(data002)
			require.NoError(t, err)

			mustbe := sha1Hash.Sum(nil)
			sha1Hash.Reset()

			rw01 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw01.Write(data001)
			require.NoError(t, err)
			rw02 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw01.Write(data002)
			require.NoError(t, err)

			err = rw01.Add(rw02)
			require.NoError(t, err)

			err = rw01.Dump(sha1Hash)
			require.NoError(t, err)

			valueIs := sha1Hash.Sum(nil)
			t.Logf("%v\n", mustbe)
			t.Logf("%v\n", valueIs)
			require.Equal(t, mustbe, valueIs)
		})

		t.Run("Three Buffers", func(t *testing.T) {
			data001 := []byte("ReaderWriter_test_0001")
			data002 := []byte("ReaderWriter_test_0002")
			data003 := []byte("ReaderWriter_test_0003")
			sha1Hash := sha1.New()
			_, err := sha1Hash.Write(data001)
			require.NoError(t, err)
			_, err = sha1Hash.Write(data002)
			require.NoError(t, err)
			_, err = sha1Hash.Write(data003)
			require.NoError(t, err)

			mustbe := sha1Hash.Sum(nil)
			sha1Hash.Reset()

			rw01 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw01.Write(data001)
			require.NoError(t, err)
			rw02 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw02.Write(data002)
			require.NoError(t, err)

			rw03 := gomessageblock.NewReaderWriterSize(128)
			_, err = rw03.Write(data003)
			require.NoError(t, err)

			err = rw01.Add(rw02)
			require.NoError(t, err)

			err = rw01.Add(rw03)
			require.NoError(t, err)

			err = rw01.Dump(sha1Hash)
			require.NoError(t, err)

			valueIs := sha1Hash.Sum(nil)
			t.Logf("%v\n", mustbe)
			t.Logf("%v\n", valueIs)
			require.Equal(t, mustbe, valueIs)
		})

	})
	t.Run("Three io.ReadeWriter Buffers", func(t *testing.T) {
		data001 := []byte("ReaderWriter_test_0001")
		data002 := []byte("ReaderWriter_test_0002")
		data003 := []byte("ReaderWriter_test_0003")
		sha1Hash := sha1.New()
		_, err := sha1Hash.Write(data001)
		require.NoError(t, err)
		_, err = sha1Hash.Write(data002)
		require.NoError(t, err)
		_, err = sha1Hash.Write(data003)
		require.NoError(t, err)

		mustbe := sha1Hash.Sum(nil)
		sha1Hash.Reset()

		rw01 := bytes.Buffer{}
		_, err = rw01.Write(data001)
		require.NoError(t, err)

		rw02 := bytes.Buffer{}
		_, err = rw02.Write(data002)
		require.NoError(t, err)

		rw03 := bytes.Buffer{}
		_, err = rw03.Write(data003)
		require.NoError(t, err)

		ma := gomessageblock.NewReaderWriter()
		err = ma.AddReaders(&rw01, &rw02, &rw03)
		require.NoError(t, err)

		err = ma.Dump(sha1Hash)
		require.NoError(t, err)

		valueIs := sha1Hash.Sum(nil)
		t.Logf("%v\n", mustbe)
		t.Logf("%v\n", valueIs)
		require.Equal(t, mustbe, valueIs)
	})
	t.Run("setnext", func(t *testing.T) {
		b1 := gomessageblock.NewReaderWriterBlock([]byte{1, 2, 3, 4, 5})
		b2 := gomessageblock.NewReaderWriterBlock([]byte{6, 7, 8, 9, 10})
		b1.SetNext(b2)
		data := make([]byte, 10)
		n, _ := b1.Read(data)
		require.Equal(t, 10, n)
		require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, data)
	})
	t.Run("setnext", func(t *testing.T) {
		b1 := gomessageblock.NewReaderWriterBlock([]byte{1, 2, 3, 4, 5})
		b2 := gomessageblock.NewReaderWriterBlock([]byte{6, 7, 8, 9, 10})
		b3 := gomessageblock.NewReaderWriterBlock([]byte{11, 12, 13, 14, 15})
		_ = b1.SetNext(b2)
		_ = b1.SetNext(b3)
		data := make([]byte, 15)
		n, _ := b1.Read(data)
		require.Equal(t, 15, n)
		require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, data)
	})
}
