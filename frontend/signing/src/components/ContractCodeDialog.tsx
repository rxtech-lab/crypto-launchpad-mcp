import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { X, Copy, FileCode } from 'lucide-react'

interface ContractCodeDialogProps {
  isOpen: boolean
  onClose: () => void
  contractCode: string
  title?: string
}

export function ContractCodeDialog({
  isOpen,
  onClose,
  contractCode,
  title = "Contract Source Code"
}: ContractCodeDialogProps) {
  const handleCopyCode = async () => {
    try {
      await navigator.clipboard.writeText(contractCode)
      // You could add a toast notification here
    } catch (error) {
      console.error('Failed to copy code:', error)
    }
  }

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/25" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-4xl transform overflow-hidden rounded-2xl bg-white text-left align-middle shadow-xl transition-all">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-blue-100 rounded-lg">
                      <FileCode className="h-6 w-6 text-blue-600" />
                    </div>
                    <Dialog.Title
                      as="h3"
                      className="text-lg font-semibold leading-6 text-gray-900"
                    >
                      {title}
                    </Dialog.Title>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={handleCopyCode}
                      className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
                      title="Copy code"
                    >
                      <Copy className="h-5 w-5" />
                    </button>
                    <button
                      onClick={onClose}
                      className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
                      title="Close dialog"
                    >
                      <X className="h-5 w-5" />
                    </button>
                  </div>
                </div>

                {/* Code Content */}
                <div className="max-h-[60vh] overflow-auto">
                  <SyntaxHighlighter
                    language="solidity"
                    style={oneDark}
                    customStyle={{
                      margin: 0,
                      borderRadius: 0,
                      fontSize: '14px',
                      lineHeight: '1.5',
                    }}
                    showLineNumbers
                    wrapLines
                    wrapLongLines
                  >
                    {contractCode}
                  </SyntaxHighlighter>
                </div>

                {/* Footer */}
                <div className="flex justify-end p-6 border-t border-gray-200">
                  <button
                    type="button"
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded-lg hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 transition-colors"
                    onClick={onClose}
                  >
                    Close
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  )
}