package db

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">>before:", account1.Balance, account2.Balance)
	//go并发处理事务
	n := 10
	amount := int64(10)
	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		//txName:储存交易名称
		//txName := fmt.Sprintf("tx: %d", i+1)
		go func() {
			//不能有testing require检查错误，因为它在不同的go例程中运行，应该在本例子使用
			//采用channel旨在链接并发的go例程，允许没有显示锁定情况下共享数据
			//开两个通道，一个接收error，另一个接收其他TransferResul
			ctx := context.Background()
			//传入带有事务名称的新上下文,将背景上下文作为父上下文,测试用
			//ctx := context.WithValue(context.Background(), txKey, txName)
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})
			//左边通道，右边发送错误方
			errs <- err
			results <- result
		}()
	}
	//检查result
	existed := make(map[int]bool)
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)
		/*
			检查transfer
		*/
		//检查传输对象不为空
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		//确定数据库创建了传输记录
		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		/*
			检查entries
		*/
		//检查fromEntry
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)
		//确定数据库创建了账户条目
		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		//检查toEntry
		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)
		//确定数据库创建了账户条目
		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		/*
				检查账户的余额
			TDD:首先编写测试以使当前代码中断，逐渐改进代码，知道测试通过
		*/
		//检查账户
		//检查钱从哪出的
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)
		//检查钱从哪进的
		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)
		//检查账户余额
		fmt.Println(">>tx:", fromAccount.Balance, toAccount.Balance) //查看每笔交易后的结果
		diff1 := account1.Balance - fromAccount.Balance              //出钱
		diff2 := toAccount.Balance - account2.Balance                //进钱
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0)
		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		//检查现有的map不应该包含k
		require.NotContains(t, existed, k)
		existed[k] = true

	}
	//检查两个账户最终更新余额，首先从数据库获取更新后的账户1和账户2
	updateAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">>last:", updateAccount1.Balance, updateAccount2.Balance)
	require.Equal(t, account1.Balance-int64(n)*amount, updateAccount1.Balance) //运算类型一致
	require.Equal(t, account2.Balance+int64(n)*amount, updateAccount2.Balance)

}

// 防止死锁最好的方法是确保程序总是以一致的顺序获取锁
// 本文方法就是按照账号小的先执行
func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">>before:", account1.Balance, account2.Balance)
	//go并发处理事务
	n := 10
	amount := int64(10)
	errs := make(chan error)
	//results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID
		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}
		go func() {
			//不能有testing require检查错误，因为它在不同的go例程中运行，应该在本例子使用
			//采用channel旨在链接并发的go例程，允许没有显示锁定情况下共享数据
			//开两个通道，一个接收error，另一个接收其他TransferResul
			ctx := context.Background()
			//传入带有事务名称的新上下文,将背景上下文作为父上下文,测试用
			//ctx := context.WithValue(context.Background(), txKey, txName)
			_, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})
			errs <- err
		}()
	}
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}
	//检查两个账户最终更新余额，首先从数据库获取更新后的账户1和账户2
	updateAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">>last:", updateAccount1.Balance, updateAccount2.Balance)
	require.Equal(t, account1.Balance, updateAccount1.Balance) //运算类型一致
	require.Equal(t, account2.Balance, updateAccount2.Balance)

}
